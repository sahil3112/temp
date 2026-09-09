#!/usr/bin/env python3

from __future__ import annotations

import yaml
from fastapi import APIRouter, Request
from fastapi.responses import JSONResponse

router = APIRouter()


def _jsonable(value):
    if isinstance(value, bytes):
        return value.decode("utf-8", "replace")
    return value


def parse_manifest(body: bytes):
    return yaml.safe_load(body)


@router.post("/api/v1/ingest")
async def ingest(request: Request):
    body = await request.body()
    if not body.strip():
        return JSONResponse(
            {"status": "rejected", "detail": "empty manifest"},
            status_code=400,
        )
    try:
        data = parse_manifest(body)
    except yaml.YAMLError:
        return JSONResponse(
            {"status": "rejected", "detail": "unsafe or invalid YAML"},
            status_code=400,
        )
    return {"status": "ingested", "ingested": _jsonable(data)}
