#!/usr/bin/env python3
from __future__ import annotations

import base64
import ctypes
import multiprocessing
import os
import tempfile
from pathlib import Path

from fastapi import APIRouter, Request
from fastapi.responses import JSONResponse

RECORD_SIZE = 64
HOOK_SIZE = 128
PAD = 4096
DEFAULT_HOOK = b"echo packed-ok"

router = APIRouter()


def native_pack(data: bytes) -> dict:
    ctx = multiprocessing.get_context("spawn")
    queue = ctx.Queue()
    proc = ctx.Process(target=_worker_main, args=(data, queue))
    proc.start()
    proc.join(timeout=5)
    if proc.is_alive():
        proc.terminate()
        proc.join(1)
        return {"status": "worker_crashed", "detail": "native packer timed out"}
    if proc.exitcode != 0:
        return {
            "status": "worker_crashed",
            "detail": f"native packer exited {proc.exitcode}",
        }
    try:
        return queue.get_nowait()
    except Exception:
        return {
            "status": "worker_crashed",
            "detail": "native packer returned no result",
        }


def _worker_main(data: bytes, queue) -> None:
    queue.put(_pack_in_worker(data))


def _pack_in_worker(data: bytes) -> dict:
    libc = ctypes.CDLL(None)
    libc.memcpy.argtypes = [ctypes.c_void_p, ctypes.c_void_p, ctypes.c_size_t]
    libc.system.argtypes = [ctypes.c_char_p]
    libc.system.restype = ctypes.c_int

    buf = ctypes.create_string_buffer(RECORD_SIZE + HOOK_SIZE + PAD)
    ctypes.memmove(ctypes.addressof(buf) + RECORD_SIZE, DEFAULT_HOOK, len(DEFAULT_HOOK))

    if data:
        src = (ctypes.c_char * len(data)).from_buffer_copy(data)
        libc.memcpy(ctypes.addressof(buf), ctypes.addressof(src), len(data))

    out_fd, out_path = tempfile.mkstemp(prefix="globo_hook_")
    saved_stdout = os.dup(1)
    try:
        os.dup2(out_fd, 1)
        os.close(out_fd)
        libc.system(ctypes.c_char_p(ctypes.addressof(buf) + RECORD_SIZE))
    finally:
        os.dup2(saved_stdout, 1)
        os.close(saved_stdout)

    hook_output = Path(out_path).read_bytes().decode("utf-8", "replace")
    try:
        os.unlink(out_path)
    except OSError:
        pass

    return {
        "status": "packed",
        "bytes_written": len(data),
        "hook_output": hook_output,
    }


@router.post("/api/v1/pack")
async def pack(request: Request):
    body = await request.json()
    raw = body.get("payload_b64")
    if not raw or not isinstance(raw, str):
        return JSONResponse(
            {"status": "rejected", "detail": "payload_b64 is required"},
            status_code=400,
        )
    try:
        data = base64.b64decode(raw)
    except Exception:
        return JSONResponse(
            {"status": "rejected", "detail": "invalid payload_b64"},
            status_code=400,
        )
    result = native_pack(data)
    code = {"packed": 200, "rejected": 400}.get(result.get("status"), 500)
    return JSONResponse(result, status_code=code)
