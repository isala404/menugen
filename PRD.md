# MenuGen Product Requirements Document (PRD)

## 1. Document Control
- Owner: Product / Engineering (initial draft by AI assistant)
- Status: Draft v0.1
- Last Updated: 2025-09-16
- Reviewers: TBD (Product, Eng Lead, Design, Data Privacy)

## 2. Executive Summary
MenuGen converts a photographed restaurant menu (printed or handwritten) into an enriched, structured, queryable "virtual menu" with dish metadata and AI‑generated dish images and descriptions. The system ingests an image, performs OCR + semantic structuring (LLM), enhances each dish (description + synthetic image), stores normalized data in Postgres, and returns a JSON representation to the client for rendering an interactive digital menu.

## 3. Goals & Non-Goals
### 3.1 Primary Goals
1. Enable a user to upload a single menu photo and receive a structured JSON menu within acceptable latency (< 60s P95 MVP).
2. Extract sections, dish names, prices, dietary indicators, and raw text reliably (≥ 85% field-level precision MVP).
3. Generate an AI description (1–2 sentences) per dish to standardize clarity.
4. Generate an illustrative (not photoreal marketing) image per dish.
5. Persist menu, dishes, and asset references in Postgres for later retrieval and refinement.
6. Provide an API for polling menu processing status and retrieving final structured output.

### 3.2 Secondary Goals (Nice-to-have if low lift)
- Ingredient-level extraction.
- Dietary tags (vegan, vegetarian, gluten-free) inference.
- Multi-image menu stitching.
- Simple versioning for re-uploads of improved photos.

### 3.3 Non-Goals (Explicitly Out of Scope for MVP)
- Real-time (sub-5s) transformation.
- Multi-language translation (English-only MVP; detection allowed for future).
- Price currency conversion.
- Nutrition calculation.
- Table ordering or payment integration.

## 4. Target Users & Personas
| Persona | Description | Primary Need |
|---------|-------------|--------------|
| Indie Restaurateur | Small venue owner lacking digital menu | Fast digitization without manual data entry |
| Menu Aggregator Ops | Internal ops team digitizing menus at scale | Repeatable structured extraction pipeline |
| Accessibility User (Downstream) | Patron using screen reader | Clean semantic dish structure |

## 5. User Stories (Prioritized)
1. As a user, I can upload a clear photo of a menu to start processing.
2. As a user, I can see the processing status (PENDING, RUNNING, COMPLETE, FAILED).
3. As a user, I can retrieve the final structured menu as JSON.
4. As a user, I receive dish names, prices, and section grouping.
5. As a user, I see a short AI-generated description per dish.
6. As a user, I see an image for each dish.
7. As a user, I can re-poll without creating duplicate menus.
8. As an internal operator, I can inspect raw OCR text for failure debugging.
9. (Stretch) As a user, I can request regeneration of a single dish image or description.

## 6. Functional Requirements
| ID | Requirement | Priority | Acceptance Criteria |
|----|-------------|----------|---------------------|
| FR1 | Upload endpoint to submit image | Must | POST /menu accepts image (JPEG/PNG ≤ 8MB) returns menu_id + status=PENDING |
| FR2 | Async processing pipeline | Must | Status transitions: PENDING→PROCESSING→COMPLETE/FAILED stored in DB |
| FR3 | OCR + LLM structuring | Must | Output schema validated; invalid -> FAILED with reason |
| FR4 | Dish enhancement (desc + image) | Must | Each dish has description + image_url (placeholder if generation fails) |
| FR5 | Poll endpoint | Must | GET /menu/{id} returns status + partial/final data |
| FR6 | Data persistence | Must | Menu, dishes, images stored in Postgres with FK consistency |
| FR7 | Idempotent upload (same hash) | Should | Same image hash returns existing menu_id (optional MVP) |
| FR8 | Basic auth / API key | Should | Requests require API key header (MVP security) |
| FR9 | Error handling | Must | Structured error codes JSON {code,message} |
| FR10 | Rate limiting | Could | 60 uploads / hour / key (deferred) |
| FR11 | Regenerate single dish asset | Could | POST /dish/{id}/regenerate (stretch) |

## 7. Non-Functional Requirements
| Category | Target |
|----------|--------|
| Performance | < 60s P95 full menu (≤ 25 dishes) |
| Availability | 99% uptime (single region) |
| Scalability | Queue-based; horizontal worker scaling |
| Security | API key; image storage access controlled (signed URLs or private bucket) |
| Privacy | Discard raw images after processing if policy requires; or retain with encrypted storage |
| Observability | Structured logs (request id), basic metrics (processing_time_ms, dishes_count) |
| Reliability | Retry transient LLM/image generation failures (max 2) |
| Cost | <$0.10 per average menu (OCR+LLM+images) target |

## 8. Success Metrics / KPIs
- Time to structured menu (median, P95).
- Extraction precision (manual sample audit). Target 85%+.
- Dish enrichment success rate (image + description) ≥ 95%.
- Failure rate < 5% of uploads.
- Average cost per menu.

## 9. Constraints & Assumptions
- Single image input MVP (no multi-page merge).
- LLM provides both OCR assistance and structuring; or OCR first (e.g., Vision model) then text structuring prompt.
- Synthetic dish images generated synchronously per dish (may extend latency) for MVP; future: batch or async second phase.
- Postgres chosen for relational integrity.
- Frontend React single-page client polls; no websockets initially.

## 10. High-Level Architecture
Frontend (React) → POST /menu (image multipart)
Backend (Go service) orchestrates:
1. Store upload (temp storage).
2. Submit image to OCR/LLM → structured draft menu JSON.
3. Normalize & persist Menu + Dishes (without enrichment fields initially?)
4. For each dish: generate description (LLM) and generate image (image model service / Fireworks).
5. Update dish rows with description + image_url.
6. Mark menu COMPLETE.
7. Poll endpoint returns cumulative progress (percentage = enriched_dishes / total_dishes).

Components:
- API Layer (Go HTTP handlers)
- Worker (could be same process goroutine pool for MVP)
- External: OpenAI (Vision + text), Fireworks (image), Object Storage (S3 or local), Postgres

## 11. Detailed Workflow
1. Client uploads image via POST /menu.
2. API validates size/content-type, creates menu row: status=PENDING.
3. Dispatch processing goroutine / queue item.
4. OCR + structuring step obtains raw text + structured candidate.
5. Validate schema (sections, dishes array). On failure: update status=FAILED(reason_code).
6. Persist normalized entities (menu, sections, dishes placeholder entries).
7. Iterate dishes: for each dish concurrently (bounded N):
   - Generate description (prompt with name + section context + price).
   - Generate image; store; capture URL.
   - Update dish row progress counter.
8. When all dishes processed: status=COMPLETE, set completed_at.
9. Poll endpoint returns status + data (if COMPLETE full; if PROCESSING partial with processed_count).

## 12. Data Model (Draft)
```
menus
  id (uuid pk)
  original_filename
  image_hash (sha256) UNIQUE? (optional MVP)
  status (enum: PENDING, PROCESSING, COMPLETE, FAILED)
  failure_reason (text nullable)
  total_dishes (int)
  processed_dishes (int)
  created_at
  updated_at
  completed_at

menu_sections
  id (uuid pk)
  menu_id (fk)
  name (text)
  position (int)

dishes
  id (uuid pk)
  menu_id (fk)
  section_id (fk nullable if ungrouped)
  name (text)
  price_cents (int nullable)
  currency (char(3) default 'USD')
  raw_price_string (text nullable)
  description (text nullable)
  image_url (text nullable)
  status (enum: PENDING, COMPLETE, FAILED)
  failure_reason (text nullable)
  position (int)
  created_at
  updated_at
```

## 13. API Design (MVP)
### 13.1 POST /menu
Request: multipart/form-data (file field: `image`)
Headers: `X-API-Key: <key>`
Response 202:
```
{ "menu_id": "uuid", "status": "PENDING" }
```
Errors: 400 (validation), 401 (auth), 415 (unsupported type), 429 (rate limit), 500.

### 13.2 GET /menu/{id}
Response 200:
```
{
  "menu_id": "uuid",
  "status": "PROCESSING",
  "progress": { "processed_dishes": 4, "total_dishes": 12 },
  "menu": { /* present only if COMPLETE or partial structure if chosen */ }
}
```
Status Codes: 200, 404, 401.

### 13.3 (Future) POST /dish/{id}/regenerate
- Body: { type: "image" | "description" | "both" }

### 13.4 Error Object
```
{ "error": { "code": "IMAGE_TOO_LARGE", "message": "..." } }
```

## 14. Prompting Strategy (Conceptual)
- OCR/Structure Prompt: Provide raw OCR tokens or pass image to vision-capable model requesting JSON schema: sections -> dishes (name, price, possible descriptors). Enforce JSON via response format instructions.
- Description Prompt: Input dish name + section + optional adjectives; limit 30 words; neutral tone.

## 15. Image Generation Strategy
- Model: Fireworks-compatible image endpoint.
- Size: 512x512 (MVP) to balance cost/time.
- Style: Clean, lightly stylized illustrative realism (avoid exaggeration); consistent background.
- Failure fallback: Generic placeholder image asset URL.

## 16. Progress Calculation
`progress = processed_dishes / total_dishes` (0–1). Update after each dish completion or failure.

## 17. Security & Privacy
- API key stored hashed (bcrypt) in DB or static env list MVP.
- Validate MIME type and magic bytes to avoid arbitrary file upload.
- Optional: purge raw original images after COMPLETE if policy requires.
- Log redaction: do not log full API key, only prefix.

## 18. Observability
Metrics (Prometheus style names example):
- `menugen_menu_process_duration_seconds` (histogram)
- `menugen_dish_enrichment_duration_seconds`
- `menugen_menu_failures_total{reason="STRUCTURE_VALIDATION"}`
Logs: JSON structured with `trace_id`, `menu_id`, `component`.
Tracing: (Future) OpenTelemetry integration.

## 19. Risks & Mitigations
| Risk | Impact | Mitigation |
|------|--------|------------|
| OCR inaccuracies on low-quality images | Incorrect dishes | Provide user guidance; confidence scoring; allow manual correction future |
| Latency spikes from image generation | Poor UX | Parallelize dish enrichment with concurrency limit + progressive polling |
| LLM JSON format drift | Pipeline failures | Strict JSON schema validation + retry with repair prompt |
| Cost overrun due to many dishes | Financial | Hard cap of 40 dishes per menu MVP; cost monitoring |
| Abuse / spam uploads | Cost & security | API keys + rate limit + image type validation |
| Image generation inappropriate content | Reputation | Use safe prompt templates + moderation checks |

## 20. Open Questions
1. Which vision model (OpenAI vs. specialized OCR) yields best accuracy/cost? Pilot needed.
2. Store generated images where? (S3 bucket vs. local dev filesystem) – decide infra.
3. Do we permit partial menu retrieval mid-processing? (Probably yes: show structure minus pending dish assets.)
4. Currency parsing rules for international menus? (Defer.)
5. Versioning strategy for regenerated images/descriptions (append history table?).

## 21. Roadmap (Indicative)
| Phase | Scope | Target |
|-------|-------|--------|
| Week 1 | Skeleton API, DB schema migration, POST/GET endpoints, basic status | Day 7 |
| Week 2 | OCR + structuring integration, schema validation | Day 14 |
| Week 3 | Dish enrichment (desc + image), progress tracking | Day 21 |
| Week 4 | Hardening (auth, logging, metrics), polish React UI display | Day 28 |
| Post-MVP | Regeneration, multi-image, translation, tagging, manual editor | Ongoing |

## 22. Acceptance Criteria Summary
MVP accepted when: user can upload a menu image and within 60s receive a COMPLETED structured menu with ≥85% extraction accuracy on test set of 20 menus; all dishes have description & image or placeholder; failure rate <5%; metrics and logs visible.

---
End of PRD.
