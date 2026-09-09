#!/usr/bin/env python3

from __future__ import annotations

from pydantic import BaseModel, ConfigDict, Field
from fastapi import APIRouter
from fastapi.responses import JSONResponse

router = APIRouter()


class Record(BaseModel):
    model_config = ConfigDict(extra="forbid")

    sku: str = Field(min_length=1, max_length=32)
    qty: int = Field(gt=0)
    price_cents: int = Field(ge=0)


class JobCreate(BaseModel):
    model_config = ConfigDict(extra="forbid")

    job_id: str = Field(min_length=1, max_length=64)
    source: str = Field(min_length=1, max_length=64)
    records: list[Record] = Field(min_length=1)
    notify_email: str | None = None


def accept_job(payload: JobCreate) -> dict:
    total_cents = sum(rec.qty * rec.price_cents for rec in payload.records)
    return {
        "status": "accepted",
        "job_id": payload.job_id,
        "source": payload.source,
        "record_count": len(payload.records),
        "accepted": payload.model_dump(),
        "total_cents": total_cents,
        
    }


def error_payload(_exc: BaseException) -> dict:
    return {"status": "error", "detail": "unable to process job"}


@router.post("/api/v1/jobs")
async def create_job(payload: JobCreate):
    try:
        return accept_job(payload)
    except Exception as exc:
        return JSONResponse(error_payload(exc), status_code=500)
