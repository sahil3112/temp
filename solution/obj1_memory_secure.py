#!/usr/bin/env python3
from __future__ import annotations

import base64

from fastapi import APIRouter, Request
from fastapi.responses import JSONResponse

RECORD_SIZE = 64

router = APIRouter()


def native_pack(data: bytes) -> dict:

    if len(data) > RECORD_SIZE:
        return {
            "status": "rejected",
            "detail": f"payload exceeds {RECORD_SIZE}-byte record",
        }
    record = bytearray(RECORD_SIZE)
    record[: len(data)] = data
    return {"status": "packed", "bytes_written": len(data)}


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
