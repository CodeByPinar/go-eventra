---
goal: Eventra platformunu auth ürününden full event SaaS platformuna dönüştürmek
version: 1.0
date_created: 2026-03-22
last_updated: 2026-03-22
owner: Eventra Core Team
status: Planned
tags: [feature, architecture, auth, events, ticketing, realtime, security, monetization]
---

# Introduction

![Status: Planned](https://img.shields.io/badge/status-Planned-blue)

Bu plan, mevcut Eventra auth altyapısını koruyarak event management, ticketing, real-time etkileşim, geliştirici API'leri, observability ve monetization katmanlarıyla production-grade bir SaaS platformuna dönüştürür.

## 1. Requirements & Constraints

- REQ-001: OAuth (Google, GitHub) ile sosyal giriş.
- REQ-002: Session/device yönetimi (aktif oturum listeleme, cihazdan çıkış).
- REQ-003: Email verification ve password reset akışları.
- REQ-004: Event CRUD, public/private, kapasite, tag/kategori, search/filter.
- REQ-005: Ticketing (free/paid), QR ticket, check-in, capacity tracking.
- REQ-006: Real-time attendee/check-in/chat/notification.
- REQ-007: Public API + API key + webhook + analytics API.
- REQ-008: Payment (Stripe veya iyzico), komisyon modeli.
- REQ-009: Organization/role/multi-tenant.
- REQ-010: UI dashboard, discover, responsive, dark mode.
- SEC-001: Login brute-force koruması sistem genelinde korunmalı ve genişletilmeli.
- SEC-002: CSRF, XSS, audit log, IP tracking zorunlu.
- SEC-003: Tenant isolation kritik (org_id bazlı erişim kontrolü).
- CON-001: Mevcut Go net/http + PostgreSQL mimarisi korunacak.
- CON-002: Migration-first gelişim; tüm şema değişiklikleri SQL migration ile yapılacak.
- CON-003: İlk versiyonda monolith-modular kalınacak, event-driven bileşenler incrementally eklenecek.
- GUD-001: Özellikler feature flag ile yayınlanmalı.
- GUD-002: Tüm yeni endpointler için integration test zorunlu.

## 2. Implementation Steps

### Implementation Phase 1 - Foundation and Domain Expansion (2-3 hafta)

- GOAL-001: Auth dışına taşan core domain'i kurmak ve event/ticket altyapısını hazırlamak.

| Task | Description | Completed | Date |
|------|-------------|-----------|------|
| TASK-001 | Eventra domain boundaries tanımla: `internal/domain/event`, `internal/domain/ticket`, `internal/domain/organization`, `internal/domain/notification` klasörlerini oluştur |  |  |
| TASK-002 | Event ve ticket için migration setlerini ekle (`configs/migrations/000004+`) |  |  |
| TASK-003 | Event repository/usecase/http handler iskeletlerini ekle (`internal/repository/postgres`, `internal/usecase/event`, `internal/delivery/httpserver`) |  |  |
| TASK-004 | API versioning stratejisi belirle (`/api/v1` korunur, yeni kaynak endpointleri eklenir) |  |  |
| TASK-005 | Search için PostgreSQL GIN/BTREE indexleri ekle (title/date/location/tags) |  |  |

### Implementation Phase 2 - Auth System Upgrade (2 hafta)

- GOAL-002: Ürün düzeyi kimlik doğrulama ve hesap güvenliği özelliklerini tamamlamak.

| Task | Description | Completed | Date |
|------|-------------|-----------|------|
| TASK-006 | OAuth provider entegrasyonu (Google, GitHub) + account linking |  |  |
| TASK-007 | Session/device modeli ekle: `user_sessions` tablosu, cihaz adı/IP/last_seen |  |  |
| TASK-008 | Email verification token akışı ve endpointleri (`/auth/verify-email`) |  |  |
| TASK-009 | Password reset request/confirm akışı ve token invalidation |  |  |
| TASK-010 | Rate limiter'ı IP+email+device fingerprint kombinasyonuna yükselt |  |  |

### Implementation Phase 3 - Event Management + Ticketing MVP (3-4 hafta)

- GOAL-003: Ürünün ana kullanım değerini veren event ve ticket akışını canlıya almak.

| Task | Description | Completed | Date |
|------|-------------|-----------|------|
| TASK-011 | Event create/update/delete/list/get endpointleri |  |  |
| TASK-012 | Public/private visibility ve organizer ownership kontrolü |  |  |
| TASK-013 | Participant limit + waitlist mantığı |  |  |
| TASK-014 | Tag/category sistemi ve faceted search/filter endpointleri |  |  |
| TASK-015 | Ticket types (free/paid), order creation, capacity decrement transaction |  |  |
| TASK-016 | QR ticket üretimi + check-in endpointi + duplicate check-in koruması |  |  |

### Implementation Phase 4 - Real-time, Notifications, Developer APIs (3 hafta)

- GOAL-004: Modern SaaS deneyimi ve geliştirici ekosistemini açmak.

| Task | Description | Completed | Date |
|------|-------------|-----------|------|
| TASK-017 | WebSocket gateway: live attendee count, live check-in updates, event chat channels |  |  |
| TASK-018 | Notification orchestration: in-app + email reminder + push (web push) |  |  |
| TASK-019 | Public API key sistemi (`api_keys` tablosu, hashed key, scope, rotation) |  |  |
| TASK-020 | Webhook delivery altyapısı (`webhook_endpoints`, `webhook_deliveries`, retry/backoff, signature) |  |  |
| TASK-021 | Analytics API (event_views, conversion_rate, attendance_rate) |  |  |

### Implementation Phase 5 - Advanced Backend and Security Hardening (2-3 hafta)

- GOAL-005: Ölçeklenebilirlik, gözlemlenebilirlik ve enterprise güvenlik standardına çıkmak.

| Task | Description | Completed | Date |
|------|-------------|-----------|------|
| TASK-022 | Event-driven omurga: başlangıçta Redis stream/NATS, sonrasında Kafka opsiyonel |  |  |
| TASK-023 | Async worker katmanı (email, webhook retry, reminder jobs, fraud checks) |  |  |
| TASK-024 | Structured logging + request correlation id + security audit enrichment |  |  |
| TASK-025 | Prometheus metrics + OpenTelemetry traces + baseline alerting |  |  |
| TASK-026 | CSRF/XSS sertleştirme, IP reputation, admin audit log ekranı |  |  |

### Implementation Phase 6 - Monetization + Multi-tenant + UI Excellence (4 hafta)

- GOAL-006: Ticari ürün kabiliyeti ve farklılaştıran kullanıcı deneyimi.

| Task | Description | Completed | Date |
|------|-------------|-----------|------|
| TASK-027 | Payment integration (Stripe öncelikli, iyzico adaptörü opsiyonel) |  |  |
| TASK-028 | Commission ve payout hesaplama modeli |  |  |
| TASK-029 | Organization/team/roles (admin-organizer-member) + tenant enforcement middleware |  |  |
| TASK-030 | Dashboard (analytics/revenue/user growth) + discover page + önerilen eventler |  |  |
| TASK-031 | Dark mode, responsive polishing, motion transitions |  |  |
| TASK-032 | PWA hazırlığı + push permission UX |  |  |

## 3. Alternatives

- ALT-001: Tüm modülleri tek sprintte geliştirmek. Reddedildi: test ve release riski çok yüksek.
- ALT-002: Tam microservice dönüşümüyle başlamak. Reddedildi: erken karmaşıklık ve operasyon maliyeti.
- ALT-003: Elasticsearch ile erken arama altyapısı kurmak. Reddedildi: önce PostgreSQL full-text + index ile doğrulama daha hızlı.

## 4. Dependencies

- DEP-001: PostgreSQL 14+ (JSONB, indexes, transactional consistency).
- DEP-002: Redis (queue/cache/rate limiting backend).
- DEP-003: OAuth credentials (Google/GitHub).
- DEP-004: SMTP/Email provider (Resend, SES, SendGrid vb.).
- DEP-005: Payment provider hesabı (Stripe veya iyzico).
- DEP-006: Metrics/Tracing stack (Prometheus + Grafana + OTEL collector).

## 5. Files

- FILE-001: `configs/migrations/*.sql` (yeni domain tabloları).
- FILE-002: `internal/domain/**` (event, ticket, organization, notification varlıkları).
- FILE-003: `internal/repository/postgres/**` (event/ticket/session/webhook repository).
- FILE-004: `internal/usecase/**` (event, ticket, payment, notification, analytics usecase).
- FILE-005: `internal/delivery/httpserver/**` (router, handler, middleware, ws handler).
- FILE-006: `internal/config/config.go` (yeni environment değişkenleri).
- FILE-007: `pkg/security/**` (api key hash/signature utilities).
- FILE-008: `frontend/src/**` (dashboard, discover, organizer panel, notification panel).

## 6. Testing

- TEST-001: Auth regression suite (register/login/refresh/logout + oauth + session revocation).
- TEST-002: Event lifecycle integration testleri (create, publish, search, filter).
- TEST-003: Ticketing transaction testleri (capacity race, duplicate check-in, payment states).
- TEST-004: Webhook retry/signature validation testleri.
- TEST-005: API key scope ve rate limit testleri.
- TEST-006: Frontend e2e testleri (critical flows: event create, buy ticket, check-in, dashboard).

## 7. Risks & Assumptions

- RISK-001: Ticket capacity yarış durumları oversell yaratabilir. Mitigasyon: DB transaction + row level lock.
- RISK-002: Real-time chat kötüye kullanım/spam riski taşır. Mitigasyon: moderation + throttling.
- RISK-003: Çok erken çoklu provider (Stripe + iyzico) scope riskini artırır. Mitigasyon: önce tek provider.
- RISK-004: Multi-tenant izolasyon hatası veri sızıntısı riski doğurur. Mitigasyon: zorunlu org_id policy + integration tests.
- ASSUMPTION-001: İlk hedef pazar web; native mobile daha sonra.
- ASSUMPTION-002: İlk sürümde tek bölge deployment kabul edilebilir.

## 8. Related Specifications / Further Reading

- `README.md`
- `api/auth.md`
- `configs/migrations/000003_security_hardening.up.sql`

## 9. Suggested Release Train

- Release A (4-5 hafta): Phase 1 + Phase 2
- Release B (4 hafta): Phase 3
- Release C (3 hafta): Phase 4
- Release D (4-5 hafta): Phase 5 + Phase 6

## 10. First Build Slice (Recommended Immediate Start)

1. Event domain + migrations + CRUD + search/filter (public/private + tags).
2. Ticket free flow + QR + check-in + live attendee count.
3. OAuth + email verification + session management.
4. Dashboard basic analytics (event views + attendance).

Bu dört adım tamamlandığında proje, auth demo seviyesinden gerçek ürün seviyesine geçer.
