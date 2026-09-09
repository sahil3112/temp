#!/usr/bin/env python3

from __future__ import annotations

import traceback
from fastapi import APIRouter, Request
from fastapi.responses import JSONResponse

router = APIRouter()

CONFIG = {
    "db_url": "postgres://globo_app:S3cret-Warehouse-Pw@db.internal.globomantics.local:5432/pipeline",
    "internal_token": "globo_live_sk_4f9c2a1b8e7d",
}


def accept_job(payload: dict) -> dict:
    job_id = payload["job_id"]
    source = payload["source"]
    records = payload["records"]
    total_cents = 0
    for rec in records:
        total_cents += rec["qty"] * rec["price_cents"]
    return {
        "status": "accepted",
        "job_id": job_id,
        "source": source,
        "record_count": len(records),
        "accepted": payload,
        "total_cents": total_cents,
        
    }


def error_payload(exc: BaseException) -> dict:
    return {
        "error": str(exc),
        "traceback": traceback.format_exc(),
        "file": __file__,
        "config": CONFIG,
    }


@router.post("/api/v1/jobs")
async def create_job(request: Request):
    try:
        payload = await request.json()
        return accept_job(payload)
    except Exception as exc:
        return JSONResponse(error_payload(exc), status_code=500)
