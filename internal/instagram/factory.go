// Package facebook — Factory tạo instance theo nền tảng (plugin registration pattern)
//
// Tránh circular import: facebook không import facebook/web/android/mfb.
// Thay vào đó, mỗi platform package tự đăng ký vào registry trong init().
// app.go blank-import các platform cần dùng để trigger init().
//
// Thêm nền tảng mới:
//  1. Tạo thư mục internal/facebook/<platform>/
//  2. Implement Registerer, Verifier, v.v.
//  3. Gọi instagram.RegisterPlatformXxx() trong init()
//  4. Blank-import trong app.go: _ "HVRIns/internal/instagram/<platform>"
package instagram

import (
	"fmt"
	"sync"
)

// Tên nền tảng — dùng làm key khi gọi New*() và khi đăng ký
const (
	PlatformAndroid    = "android"
	PlatformWeb        = "web"
	PlatformMfb        = "mfb"
	PlatformChrome     = "chrome"
	PlatformWebAndroid = "webandroid"
	PlatformWebRequest = "webrequest"
	PlatformIOS        = "ios"
	PlatformS23        = "s23"
	PlatformS22        = "s22"       // Samsung S22 — alias S23 flow, khác device model
	PlatformS24        = "s24"       // Samsung S24 — alias S23 flow, khác device model
	PlatformS25        = "s25"       // Samsung S25 — alias S23 flow, khác device model
	PlatformS26        = "s26"       // Samsung S26 — alias S23 flow, khác device model
	PlatformS545       = "s545"      // Samsung S23 + FB API 545
	PlatformS546       = "s546"      // Samsung S23 + FB API 546
	PlatformS547       = "s547"      // Samsung S23 + FB API 547
	PlatformS548       = "s548"      // Samsung S23 + FB API 548
	PlatformS549       = "s549"      // Samsung S23 + FB API 549
	PlatformS550       = "s550"      // Samsung S23 + FB API 550
	PlatformS551       = "s551"      // Samsung S23 + FB API 551
	PlatformS552       = "s552"      // Samsung S23 + FB API 552
	PlatformS553       = "s553"      // Samsung S23 + FB API 553
	PlatformS554       = "s554"      // Samsung S23 + FB API 554
	PlatformS553V2     = "s553v2"    // Samsung S23 + FB API 553 v2 — captured traffic mới, is_push_on=false
	PlatformS554V2     = "s554v2"    // Samsung S23 + FB API 554 v2 — captured traffic mới, is_push_on=false
	PlatformS551V2     = "s551v2"    // Samsung S23 + FB API 551 v2 — captured traffic mới, is_push_on=true
	PlatformS552V2     = "s552v2"    // Samsung S23 + FB API 552 v2 — captured traffic mới, is_push_on=true
	PlatformS550V2     = "s550v2"    // Samsung S23 + FB API 550 v2 — captured traffic mới, is_push_on=true
	PlatformS415       = "s415"      // Samsung S23 + FB API 415 — cấu trúc giống s560v2, doc_id/bloks_ver/styles_id v415
	PlatformS425       = "s425"      // Samsung S23 + FB API 425 — cấu trúc giống s560v2, doc_id/bloks_ver/styles_id v425
	PlatformS435       = "s435"      // Samsung S23 + FB API 435 — cấu trúc giống s560v2, doc_id/bloks_ver/styles_id v435
	PlatformS445       = "s445"      // Samsung S23 + FB API 445 — cấu trúc giống s560v2, doc_id/bloks_ver/styles_id v445
	PlatformS416       = "s416"      // Samsung S23 + FB API 416 — reg-only, kế thừa cấu trúc/doc_id từ s415
	PlatformS417       = "s417"      // Samsung S23 + FB API 417 — reg-only, kế thừa cấu trúc/doc_id từ s415
	PlatformS418       = "s418"      // Samsung S23 + FB API 418 — reg-only, kế thừa cấu trúc/doc_id từ s415
	PlatformS419       = "s419"      // Samsung S23 + FB API 419 — reg-only, kế thừa cấu trúc/doc_id từ s415
	PlatformS420       = "s420"      // Samsung S23 + FB API 420 — reg-only, kế thừa cấu trúc/doc_id từ s425
	PlatformS421       = "s421"      // Samsung S23 + FB API 421 — reg-only, kế thừa cấu trúc/doc_id từ s425
	PlatformS422       = "s422"      // Samsung S23 + FB API 422 — reg-only, kế thừa cấu trúc/doc_id từ s425
	PlatformS423       = "s423"      // Samsung S23 + FB API 423 — reg-only, kế thừa cấu trúc/doc_id từ s425
	PlatformS424       = "s424"      // Samsung S23 + FB API 424 — reg-only, kế thừa cấu trúc/doc_id từ s425
	PlatformS426       = "s426"      // Samsung S23 + FB API 426 — reg-only, kế thừa cấu trúc/doc_id từ s425
	PlatformS427       = "s427"      // Samsung S23 + FB API 427 — reg-only, kế thừa cấu trúc/doc_id từ s425
	PlatformS428       = "s428"      // Samsung S23 + FB API 428 — reg-only, kế thừa cấu trúc/doc_id từ s425
	PlatformS429       = "s429"      // Samsung S23 + FB API 429 — reg-only, kế thừa cấu trúc/doc_id từ s425
	PlatformS430       = "s430"      // Samsung S23 + FB API 430 — reg-only, kế thừa cấu trúc/doc_id từ s435
	PlatformS431       = "s431"      // Samsung S23 + FB API 431 — reg-only, kế thừa cấu trúc/doc_id từ s435
	PlatformS432       = "s432"      // Samsung S23 + FB API 432 — reg-only, kế thừa cấu trúc/doc_id từ s435
	PlatformS433       = "s433"      // Samsung S23 + FB API 433 — reg-only, kế thừa cấu trúc/doc_id từ s435
	PlatformS434       = "s434"      // Samsung S23 + FB API 434 — reg-only, kế thừa cấu trúc/doc_id từ s435
	PlatformS436       = "s436"      // Samsung S23 + FB API 436 — reg-only, kế thừa cấu trúc/doc_id từ s435
	PlatformS437       = "s437"      // Samsung S23 + FB API 437 — reg-only, kế thừa cấu trúc/doc_id từ s435
	PlatformS438       = "s438"      // Samsung S23 + FB API 438 — reg-only, kế thừa cấu trúc/doc_id từ s435
	PlatformS439       = "s439"      // Samsung S23 + FB API 439 — reg-only, kế thừa cấu trúc/doc_id từ s435
	PlatformS440       = "s440"      // Samsung S23 + FB API 440 — reg-only, kế thừa cấu trúc/doc_id từ s445
	PlatformS441       = "s441"      // Samsung S23 + FB API 441 — reg-only, kế thừa cấu trúc/doc_id từ s445
	PlatformS442       = "s442"      // Samsung S23 + FB API 442 — reg-only, kế thừa cấu trúc/doc_id từ s445
	PlatformS443       = "s443"      // Samsung S23 + FB API 443 — reg-only, kế thừa cấu trúc/doc_id từ s445
	PlatformS444       = "s444"      // Samsung S23 + FB API 444 — reg-only, kế thừa cấu trúc/doc_id từ s445
	PlatformS446       = "s446"      // Samsung S23 + FB API 446 — reg-only, kế thừa cấu trúc/doc_id từ s445
	PlatformS447       = "s447"      // Samsung S23 + FB API 447 — reg-only, kế thừa cấu trúc/doc_id từ s445
	PlatformS448       = "s448"      // Samsung S23 + FB API 448 — reg-only, kế thừa cấu trúc/doc_id từ s445
	PlatformS449       = "s449"      // Samsung S23 + FB API 449 — reg-only, kế thừa cấu trúc/doc_id từ s445
	PlatformS450       = "s450"      // Samsung S23 + FB API 450 — reg-only, kế thừa cấu trúc/doc_id từ s455
	PlatformS451       = "s451"      // Samsung S23 + FB API 451 — reg-only, kế thừa cấu trúc/doc_id từ s455
	PlatformS452       = "s452"      // Samsung S23 + FB API 452 — reg-only, kế thừa cấu trúc/doc_id từ s455
	PlatformS453       = "s453"      // Samsung S23 + FB API 453 — reg-only, kế thừa cấu trúc/doc_id từ s455
	PlatformS454       = "s454"      // Samsung S23 + FB API 454 — reg-only, kế thừa cấu trúc/doc_id từ s455
	PlatformS455       = "s455"      // Samsung S23 + FB API 455 — captured traffic (doc_id/bloks_ver/styles_id v455)
	PlatformS456       = "s456"      // Samsung S23 + FB API 456 — reg-only, synthetic (kế thừa docid/bloks/styles từ s455)
	PlatformS457       = "s457"      // Samsung S23 + FB API 457 — reg-only, synthetic (kế thừa docid/bloks/styles từ s455)
	PlatformS458       = "s458"      // Samsung S23 + FB API 458 — reg-only, synthetic (kế thừa docid/bloks/styles từ s455)
	PlatformS459       = "s459"      // Samsung S23 + FB API 459 — reg-only, synthetic (kế thừa docid/bloks/styles từ s455)
	PlatformS460       = "s460"      // Samsung S23 + FB API 460 — reg-only, synthetic (kế thừa docid/bloks/styles từ s465)
	PlatformS461       = "s461"      // Samsung S23 + FB API 461 — reg-only, synthetic (kế thừa docid/bloks/styles từ s465)
	PlatformS462       = "s462"      // Samsung S23 + FB API 462 — reg-only, synthetic (kế thừa docid/bloks/styles từ s465)
	PlatformS463       = "s463"      // Samsung S23 + FB API 463 — reg-only, synthetic (kế thừa docid/bloks/styles từ s465)
	PlatformS464       = "s464"      // Samsung S23 + FB API 464 — reg-only, synthetic (kế thừa docid/bloks/styles từ s465)
	PlatformS465       = "s465"      // Samsung S23 + FB API 465 — captured traffic
	PlatformS466       = "s466"      // Samsung S23 + FB API 466 — reg-only, synthetic (kế thừa docid/bloks từ s465)
	PlatformS467       = "s467"      // Samsung S23 + FB API 467 — reg-only, synthetic (kế thừa docid/bloks từ s465)
	PlatformS468       = "s468"      // Samsung S23 + FB API 468 — reg-only, synthetic (kế thừa docid/bloks từ s465)
	PlatformS469       = "s469"      // Samsung S23 + FB API 469 — reg-only, synthetic (kế thừa docid/bloks từ s465)
	PlatformS470       = "s470"      // Samsung S23 + FB API 470 — reg-only, synthetic (kế thừa docid/bloks từ s475)
	PlatformS471       = "s471"      // Samsung S23 + FB API 471 — reg-only, synthetic (kế thừa docid/bloks từ s475)
	PlatformS472       = "s472"      // Samsung S23 + FB API 472 — reg-only, synthetic (kế thừa docid/bloks từ s475)
	PlatformS473       = "s473"      // Samsung S23 + FB API 473 — reg-only, synthetic (kế thừa docid/bloks từ s475)
	PlatformS474       = "s474"      // Samsung S23 + FB API 474 — reg-only, synthetic (kế thừa docid/bloks từ s475)
	PlatformS475       = "s475"      // Samsung S23 + FB API 475 — captured traffic
	PlatformS476       = "s476"      // Samsung S23 + FB API 476 — reg-only, synthetic (kế thừa docid/bloks từ s475)
	PlatformS477       = "s477"      // Samsung S23 + FB API 477 — reg-only, synthetic (kế thừa docid/bloks từ s475)
	PlatformS478       = "s478"      // Samsung S23 + FB API 478 — reg-only, synthetic (kế thừa docid/bloks từ s475)
	PlatformS479       = "s479"      // Samsung S23 + FB API 479 — reg-only, synthetic (kế thừa docid/bloks từ s475)
	PlatformS480       = "s480"      // Samsung S23 + FB API 480 — reg-only, synthetic (kế thừa docid/bloks từ s485)
	PlatformS481       = "s481"      // Samsung S23 + FB API 481 — reg-only, synthetic (kế thừa docid/bloks từ s485)
	PlatformS482       = "s482"      // Samsung S23 + FB API 482 — reg-only, synthetic (kế thừa docid/bloks từ s485)
	PlatformS483       = "s483"      // Samsung S23 + FB API 483 — reg-only, synthetic (kế thừa docid/bloks từ s485)
	PlatformS484       = "s484"      // Samsung S23 + FB API 484 — reg-only, synthetic (kế thừa docid/bloks từ s485)
	PlatformS485       = "s485"      // Samsung S23 + FB API 485 — captured traffic
	PlatformS486       = "s486"      // Samsung S23 + FB API 486 — reg-only, synthetic (kế thừa docid/bloks từ s485)
	PlatformS487       = "s487"      // Samsung S23 + FB API 487 — reg-only, synthetic (kế thừa docid/bloks từ s485)
	PlatformS488       = "s488"      // Samsung S23 + FB API 488 — reg-only, synthetic (kế thừa docid/bloks từ s485)
	PlatformS489       = "s489"      // Samsung S23 + FB API 489 — reg-only, synthetic (kế thừa docid/bloks từ s485)
	PlatformS490       = "s490"      // Samsung S23 + FB API 490 — reg-only, synthetic (kế thừa docid/bloks từ s495)
	PlatformS491       = "s491"      // Samsung S23 + FB API 491 — reg-only, synthetic (kế thừa docid/bloks từ s495)
	PlatformS492       = "s492"      // Samsung S23 + FB API 492 — reg-only, synthetic (kế thừa docid/bloks từ s495)
	PlatformS493       = "s493"      // Samsung S23 + FB API 493 — reg-only, synthetic (kế thừa docid/bloks từ s495)
	PlatformS494       = "s494"      // Samsung S23 + FB API 494 — reg-only, synthetic (kế thừa docid/bloks từ s495)
	PlatformS496       = "s496"      // Samsung S23 + FB API 496 — reg-only, synthetic (kế thừa docid/bloks từ s495)
	PlatformS497       = "s497"      // Samsung S23 + FB API 497 — reg-only, synthetic (kế thừa docid/bloks từ s495)
	PlatformS498       = "s498"      // Samsung S23 + FB API 498 — reg-only, synthetic (kế thừa docid/bloks từ s495)
	PlatformS499       = "s499"      // Samsung S23 + FB API 499 — reg-only, synthetic (kế thừa docid/bloks từ s495)
	PlatformS495       = "s495"      // Samsung S23 + FB API 495 — captured traffic, reg-only (doc_id/bloks_ver/styles_id v495)
	PlatformS557       = "s557"      // Samsung S23 + FB API 557 — bloks/doc_id/headers mới
	PlatformS555       = "s555"      // Samsung S23 + FB API 555 — bloks/doc_id v555
	PlatformS555V2     = "s555v2"    // Samsung S23 + FB API 555 v2 — captured traffic mới, is_push_on=false
	PlatformS556       = "s556"      // Samsung S23 + FB API 556 — bloks/doc_id v556
	PlatformS558       = "s558"      // Samsung S23 + FB API 558 — integrity-machine-id, si_device_param_network_info
	PlatformS558V2     = "s558v2"    // Samsung S23 + FB API 558 v2 — captured traffic mới, is_push_on=false
	PlatformS556V2     = "s556v2"    // Samsung S23 + FB API 556 v2 — captured traffic mới, is_push_on=false
	PlatformS557V2     = "s557v2"    // Samsung S23 + FB API 557 v2 — captured traffic mới, is_push_on=false
	PlatformS559       = "s559"      // Samsung S23 + FB API 559 — bloks/doc_id mới, x-zero-eh/rmd, is_push_on=false
	PlatformS559V2     = "s559v2"    // Samsung S23 + FB API 559 v2 — captured May 2026 body/nt_context
	PlatformS560       = "s560"      // Samsung S23 + FB API 560 — bloks/doc_id mới, logic identical s559
	PlatformS560V2     = "s560v2"    // Samsung S23 + FB API 560 v2 - captured May 2026 reg body, verify upgraded from s559v2
	PlatformS560V3     = "s560v3"    // Samsung S23 + FB API 560 v3 - verify only, API spec copied from s560v2
	PlatformS561       = "s561"      // Samsung S23 + FB API 561
	PlatformS561V2     = "s561v2"    // Samsung S23 + FB API 561 v2 — FBAV/561.0.0.42.67, doc_id mới
	PlatformS561V3     = "s561v3"    // Samsung S23 + FB API 561 v3 — headers mới (bỏ zero-*, reorder)
	PlatformS561V99    = "s561v99"   // Samsung S23 + FB API 561 v3 — Type Reg 2 (step-by-step, tách riêng)
	PlatformS561V4S21  = "s561v4s21" // Samsung Galaxy S21+ (SM-G996B) + FB API 561 v2 — capture mới, FBBV 976056141, theme XMDS+FDS
	PlatformS561V4S23  = "s561v4s23" // Samsung S23 (SM-S911B) + FB API 561 v2 — capture mới, FBBV 976056141, theme XMDS+FDS
	PlatformS562       = "s562"      // Samsung S23 + FB API 562
	PlatformS562V3     = "s562v3"    // Samsung S23 + FB API 562 v3 — captured traffic mới
	PlatformS562V4S21  = "s562v4s21" // Samsung Galaxy S21+ (SM-G996B) + FB API 562 v4 — capture FBAV 562.0.0.51.73, FBBV 976057955
	PlatformS562V4S23  = "s562v4s23" // Samsung Galaxy S23 (SM-S911B) + FB API 562 v4 — capture FBAV 562.0.0.51.73, FBBV 976057955
	PlatformS563       = "s563"      // Samsung S23 + FB API 563 — logic identical s559
	PlatformS563V2     = "s563v2"    // Samsung S23 + FB API 563 v2 — build 563.0.0.0.26 (new doc_id/bloks_ver, FBBV 972941018)
	PlatformS563S21    = "s563s21"   // Samsung Galaxy S21+ (SM-G996B) + FB API 563 — build .26, device S21+ thay S23
	PlatformS563V3S21  = "s563v3s21" // Samsung Galaxy S21+ (SM-G996B) + FB API 563 v3 — capture FBAV 563.0.0.0.48, FBBV 974373688, doc_id mới
	PlatformS563V4S21  = "s563v4s21" // Samsung Galaxy S21+ (SM-G996B) + FB API 563 v4 — capture FBAV 563.0.0.23.73, FBBV 978036554, doc_id mới
	PlatformS563V4S23  = "s563v4s23" // Samsung Galaxy S23 (SM-S911B) + FB API 563 v4 — same constants as s563v4s21, device S23 (density 3.0, 1080x2340)
	PlatformS563V5S21  = "s563v5s21" // Samsung Galaxy S21+ (SM-G996B) + FB API 563 v5 — capture FBAV 563.0.0.23.73, FBBV 980389559, theme FDS only
	PlatformS563V5S23  = "s563v5s23" // Samsung Galaxy S23 (SM-S911B) + FB API 563 v5 — same constants as s563v5s21, device S23 (density 3.0, 1080x2340)
	PlatformS563V6S21  = "s563v6s21" // Samsung Galaxy S21+ (SM-G996B) + FB API 563 v6 — capture FBAV 563.1.0.50.73, FBBV 986611012, doc_id …453023, bloks f9d474f2 (= v5)
	PlatformS563V6S23  = "s563v6s23" // Samsung Galaxy S23 (SM-S911B) + FB API 563 v6 — same constants as s563v6s21, device S23 (density 3.0, 1080x2340)
	PlatformS564V1S21  = "s564v1s21" // Samsung Galaxy S21+ (SM-G996B) + FB API 564 v1 — capture FBAV 564.0.0.0.17, FBBV 977893103, doc_id mới
	PlatformS564V1S23  = "s564v1s23" // Samsung Galaxy S23 (SM-S911B) + FB API 564 v1 — same constants as s564v1s21, device S23 (density 3.0, 1080x2340)
	PlatformS564V2S21  = "s564v2s21" // Samsung Galaxy S21+ (SM-G996B) + FB API 564 v2 — capture FBAV 564.0.0.0.61, FBBV 980390555, theme FDS only
	PlatformS564V2S23  = "s564v2s23" // Samsung Galaxy S23 (SM-S911B) + FB API 564 v2 — same constants as s564v2s21, device S23 (density 3.0, 1080x2340)
	PlatformS564V3S21  = "s564v3s21" // Samsung Galaxy S21+ (SM-G996B) + FB API 564 v3 — capture FBAV 564.0.0.48.74, FBBV 986612294, doc_id …598103, bloks ebb84a (= v2)
	PlatformS564V3S23  = "s564v3s23" // Samsung Galaxy S23 (SM-S911B) + FB API 564 v3 — same constants as s564v3s21, device S23 (density 3.0, 1080x2340)
	PlatformS565S21    = "s565s21"   // Samsung Galaxy S21+ (SM-G996B) + FB API 565 — FBAV 565.0.0.0.28, FBBV 984080529, density 2.8125, 1080x2400
	PlatformS565S23    = "s565s23"   // Samsung Galaxy S23 (SM-S911B) + FB API 565 — same constants as s565, device S23 (density 3.0, 1080x2340)
	PlatformS565V2S21  = "s565v2s21" // Samsung Galaxy S21+ (SM-G996B) + FB API 565 v2 — capture FBAV 565.0.0.0.58, FBBV 986097483, doc_id+bloks_ver mới, density 2.8125, 1080x2400
	PlatformS565V2S23  = "s565v2s23" // Samsung Galaxy S23 (SM-S911B) + FB API 565 v2 — same constants as s565v2s21, device S23 (density 3.0, 1080x2340)
	PlatformS399       = "s399"      // Samsung S23 + FB API 399 — flow CŨ FB4A native (POST /app/users + /auth/login, KHÔNG dùng Bloks)
	PlatformS273       = "s273"      // Vivo V2242A + FB API 273 — b-api.facebook.com/method/user.xxx, Android 9, FBAV/273.0.0.39.123
	PlatformToken      = "token"     // Token API verify — gọi trực tiếp Graph API qua access_token
	PlatformAppMessV3  = "appmessv3" // Messenger (Orca) v529 — Ver login kiểu V3 (send_login_request facebook_local_auth → EAAD + cookies) rồi add email/confirm
	PlatformAppMV3     = "appmv3reg" // Messenger (Orca) v530 — REG flow (create.account single-shot, reg→ver liền mạch)
	// Messenger (Orca) version mới — CÙNG flow appmessv3, chỉ khác doc_id/bloks/FBAV (capture FlowRegVerFb_AppMess).
	PlatformAppMV3_535    = "appmv3reg535"  // Messenger Orca 535 — REG
	PlatformAppMV3_545    = "appmv3reg545"  // Messenger Orca 545 — REG
	PlatformAppMV3_555    = "appmv3reg555"  // Messenger Orca 555 — REG
	PlatformAppMV3_563    = "appmv3reg563"  // Messenger Orca 563 — REG
	PlatformAppMV3_564    = "appmv3reg564"  // Messenger Orca 564 — REG
	PlatformAppMV3_565    = "appmv3reg565"  // Messenger Orca 565 — REG
	PlatformAppMV3_525    = "appmv3reg525"  // Messenger Orca 525 — REG
	PlatformAppMV3_515    = "appmv3reg515"  // Messenger Orca 515 — REG
	PlatformAppMV3_505    = "appmv3reg505"  // Messenger Orca 505 — REG
	PlatformAppMV3_490    = "appmv3reg490"  // Messenger Orca 490 — REG
	PlatformAppMessV3_535 = "appmessv3_535" // Messenger Orca 535 — VERIFY
	PlatformAppMessV3_545 = "appmessv3_545" // Messenger Orca 545 — VERIFY
	PlatformAppMessV3_555 = "appmessv3_555" // Messenger Orca 555 — VERIFY
	PlatformAppMessV3_563 = "appmessv3_563" // Messenger Orca 563 — VERIFY
	PlatformAppMessV3_564 = "appmessv3_564" // Messenger Orca 564 — VERIFY
	PlatformAppMessV3_565 = "appmessv3_565" // Messenger Orca 565 — VERIFY
	PlatformAppMessV3_525 = "appmessv3_525" // Messenger Orca 525 — VERIFY
	PlatformAppMessV3_515 = "appmessv3_515" // Messenger Orca 515 — VERIFY
	PlatformAppMessV3_505 = "appmessv3_505" // Messenger Orca 505 — VERIFY
	PlatformAppMessV3_490 = "appmessv3_490" // Messenger Orca 490 — VERIFY
	PlatformIOSMessReg = "iosmessreg" // Messenger Lite iOS (FBAV/563) — REG create.account (create-only, trả crypted_user_id)
	PlatformIOSMess    = "iosmess"    // Messenger Lite iOS (FBAV/563) — VERIFY add-mail + OTP confirm + live/die (reuse mail từ reg)

	// ─── iOS Native App (FBIOS) ──────────────────────────────────────────────
	// KHÁC PlatformIOS (= "ios" = iPhone Mobile Safari qua m.facebook.com).
	// graph.facebook.com/graphql + OAuth 6628568379 + FBAN/FBIOS.
	PlatformIOS562 = "ios562" // iPhone + FB iOS app (FBIOS) API 562 — multi-round nosess handshake, reg+ver
	PlatformIOS563 = "ios563" // iPhone + FB iOS app (FBIOS) API 563 — single-shot, v563 constants
	PlatformIOS564 = "ios564" // iPhone + FB iOS app (FBIOS) API 564 — multi-round nosess handshake, reg+ver (clone 555)
	PlatformIOS555 = "ios555" // iPhone + FB iOS app (FBIOS) API 555 — multi-round nosess handshake, reg+ver (clone 562)
	PlatformIOS550 = "ios550" // iPhone + FB iOS app (FBIOS) API 550 — multi-round nosess handshake, reg+ver (clone 562)
	PlatformIOS540 = "ios540" // iPhone + FB iOS app (FBIOS) API 540 — multi-round nosess handshake, reg+ver (clone 562)
	PlatformIOS530 = "ios530" // iPhone + FB iOS app (FBIOS) API 530 — multi-round nosess handshake, reg+ver (clone 562)
	PlatformIOS520 = "ios520" // iPhone + FB iOS app (FBIOS) API 520 — multi-round nosess handshake, reg+ver (clone 562)
	PlatformIOS510 = "ios510" // iPhone + FB iOS app (FBIOS) API 510 — multi-round nosess handshake, reg+ver (clone 562)
	PlatformIOS500 = "ios500" // iPhone + FB iOS app (FBIOS) API 500 — multi-round nosess handshake, reg+ver (clone 562)
	PlatformIOS490 = "ios490" // iPhone + FB iOS app (FBIOS) API 490 — multi-round nosess handshake, reg+ver (clone 562)
	PlatformIOS480 = "ios480" // iPhone + FB iOS app (FBIOS) API 480 — multi-round nosess handshake, reg+ver (clone 562)
	PlatformIOS470 = "ios470" // iPhone + FB iOS app (FBIOS) API 470 — multi-round nosess handshake, reg+ver (clone 562)
	PlatformIOS460 = "ios460" // iPhone + FB iOS app (FBIOS) API 460 — multi-round nosess handshake, reg+ver (clone 562)
	PlatformIOS450 = "ios450" // iPhone + FB iOS app (FBIOS) API 450 — multi-round nosess handshake, reg+ver (clone 562)
	PlatformIOS421 = "ios421" // iPhone + FB iOS app (FBIOS) API 421 — clone ios420
	PlatformIOS422 = "ios422" // iPhone + FB iOS app (FBIOS) API 422 — clone ios420
	PlatformIOS423 = "ios423" // iPhone + FB iOS app (FBIOS) API 423 — clone ios420
	PlatformIOS424 = "ios424" // iPhone + FB iOS app (FBIOS) API 424 — clone ios420
	PlatformIOS425 = "ios425" // iPhone + FB iOS app (FBIOS) API 425 — clone ios420
	PlatformIOS426 = "ios426" // iPhone + FB iOS app (FBIOS) API 426 — clone ios420
	PlatformIOS427 = "ios427" // iPhone + FB iOS app (FBIOS) API 427 — clone ios420
	PlatformIOS428 = "ios428" // iPhone + FB iOS app (FBIOS) API 428 — clone ios420
	PlatformIOS429 = "ios429" // iPhone + FB iOS app (FBIOS) API 429 — clone ios420
	PlatformIOS431 = "ios431" // iPhone + FB iOS app (FBIOS) API 431 — clone ios430
	PlatformIOS432 = "ios432" // iPhone + FB iOS app (FBIOS) API 432 — clone ios430
	PlatformIOS433 = "ios433" // iPhone + FB iOS app (FBIOS) API 433 — clone ios430
	PlatformIOS434 = "ios434" // iPhone + FB iOS app (FBIOS) API 434 — clone ios430
	PlatformIOS435 = "ios435" // iPhone + FB iOS app (FBIOS) API 435 — clone ios430
	PlatformIOS436 = "ios436" // iPhone + FB iOS app (FBIOS) API 436 — clone ios430
	PlatformIOS437 = "ios437" // iPhone + FB iOS app (FBIOS) API 437 — clone ios430
	PlatformIOS438 = "ios438" // iPhone + FB iOS app (FBIOS) API 438 — clone ios430
	PlatformIOS439 = "ios439" // iPhone + FB iOS app (FBIOS) API 439 — clone ios430
	PlatformIOS441 = "ios441" // iPhone + FB iOS app (FBIOS) API 441 — clone ios440
	PlatformIOS442 = "ios442" // iPhone + FB iOS app (FBIOS) API 442 — clone ios440
	PlatformIOS443 = "ios443" // iPhone + FB iOS app (FBIOS) API 443 — clone ios440
	PlatformIOS444 = "ios444" // iPhone + FB iOS app (FBIOS) API 444 — clone ios440
	PlatformIOS445 = "ios445" // iPhone + FB iOS app (FBIOS) API 445 — clone ios440
	PlatformIOS446 = "ios446" // iPhone + FB iOS app (FBIOS) API 446 — clone ios440
	PlatformIOS447 = "ios447" // iPhone + FB iOS app (FBIOS) API 447 — clone ios440
	PlatformIOS448 = "ios448" // iPhone + FB iOS app (FBIOS) API 448 — clone ios440
	PlatformIOS449 = "ios449" // iPhone + FB iOS app (FBIOS) API 449 — clone ios440
	PlatformIOS451 = "ios451" // iPhone + FB iOS app (FBIOS) API 451 — clone ios450
	PlatformIOS452 = "ios452" // iPhone + FB iOS app (FBIOS) API 452 — clone ios450
	PlatformIOS453 = "ios453" // iPhone + FB iOS app (FBIOS) API 453 — clone ios450
	PlatformIOS454 = "ios454" // iPhone + FB iOS app (FBIOS) API 454 — clone ios450
	PlatformIOS455 = "ios455" // iPhone + FB iOS app (FBIOS) API 455 — clone ios450
	PlatformIOS456 = "ios456" // iPhone + FB iOS app (FBIOS) API 456 — clone ios450
	PlatformIOS457 = "ios457" // iPhone + FB iOS app (FBIOS) API 457 — clone ios450
	PlatformIOS458 = "ios458" // iPhone + FB iOS app (FBIOS) API 458 — clone ios450
	PlatformIOS459 = "ios459" // iPhone + FB iOS app (FBIOS) API 459 — clone ios450
	PlatformIOS461 = "ios461" // iPhone + FB iOS app (FBIOS) API 461 — clone ios460
	PlatformIOS462 = "ios462" // iPhone + FB iOS app (FBIOS) API 462 — clone ios460
	PlatformIOS463 = "ios463" // iPhone + FB iOS app (FBIOS) API 463 — clone ios460
	PlatformIOS464 = "ios464" // iPhone + FB iOS app (FBIOS) API 464 — clone ios460
	PlatformIOS465 = "ios465" // iPhone + FB iOS app (FBIOS) API 465 — clone ios460
	PlatformIOS466 = "ios466" // iPhone + FB iOS app (FBIOS) API 466 — clone ios460
	PlatformIOS467 = "ios467" // iPhone + FB iOS app (FBIOS) API 467 — clone ios460
	PlatformIOS468 = "ios468" // iPhone + FB iOS app (FBIOS) API 468 — clone ios460
	PlatformIOS469 = "ios469" // iPhone + FB iOS app (FBIOS) API 469 — clone ios460
	PlatformIOS471 = "ios471" // iPhone + FB iOS app (FBIOS) API 471 — clone ios470
	PlatformIOS472 = "ios472" // iPhone + FB iOS app (FBIOS) API 472 — clone ios470
	PlatformIOS473 = "ios473" // iPhone + FB iOS app (FBIOS) API 473 — clone ios470
	PlatformIOS474 = "ios474" // iPhone + FB iOS app (FBIOS) API 474 — clone ios470
	PlatformIOS475 = "ios475" // iPhone + FB iOS app (FBIOS) API 475 — clone ios470
	PlatformIOS476 = "ios476" // iPhone + FB iOS app (FBIOS) API 476 — clone ios470
	PlatformIOS477 = "ios477" // iPhone + FB iOS app (FBIOS) API 477 — clone ios470
	PlatformIOS478 = "ios478" // iPhone + FB iOS app (FBIOS) API 478 — clone ios470
	PlatformIOS479 = "ios479" // iPhone + FB iOS app (FBIOS) API 479 — clone ios470
	PlatformIOS481 = "ios481" // iPhone + FB iOS app (FBIOS) API 481 — clone ios480
	PlatformIOS482 = "ios482" // iPhone + FB iOS app (FBIOS) API 482 — clone ios480
	PlatformIOS483 = "ios483" // iPhone + FB iOS app (FBIOS) API 483 — clone ios480
	PlatformIOS484 = "ios484" // iPhone + FB iOS app (FBIOS) API 484 — clone ios480
	PlatformIOS485 = "ios485" // iPhone + FB iOS app (FBIOS) API 485 — clone ios480
	PlatformIOS486 = "ios486" // iPhone + FB iOS app (FBIOS) API 486 — clone ios480
	PlatformIOS487 = "ios487" // iPhone + FB iOS app (FBIOS) API 487 — clone ios480
	PlatformIOS488 = "ios488" // iPhone + FB iOS app (FBIOS) API 488 — clone ios480
	PlatformIOS489 = "ios489" // iPhone + FB iOS app (FBIOS) API 489 — clone ios480
	PlatformIOS491 = "ios491" // iPhone + FB iOS app (FBIOS) API 491 — clone ios490
	PlatformIOS492 = "ios492" // iPhone + FB iOS app (FBIOS) API 492 — clone ios490
	PlatformIOS493 = "ios493" // iPhone + FB iOS app (FBIOS) API 493 — clone ios490
	PlatformIOS494 = "ios494" // iPhone + FB iOS app (FBIOS) API 494 — clone ios490
	PlatformIOS495 = "ios495" // iPhone + FB iOS app (FBIOS) API 495 — clone ios490
	PlatformIOS496 = "ios496" // iPhone + FB iOS app (FBIOS) API 496 — clone ios490
	PlatformIOS497 = "ios497" // iPhone + FB iOS app (FBIOS) API 497 — clone ios490
	PlatformIOS498 = "ios498" // iPhone + FB iOS app (FBIOS) API 498 — clone ios490
	PlatformIOS499 = "ios499" // iPhone + FB iOS app (FBIOS) API 499 — clone ios490
	PlatformIOS501 = "ios501" // iPhone + FB iOS app (FBIOS) API 501 — clone ios500
	PlatformIOS502 = "ios502" // iPhone + FB iOS app (FBIOS) API 502 — clone ios500
	PlatformIOS503 = "ios503" // iPhone + FB iOS app (FBIOS) API 503 — clone ios500
	PlatformIOS504 = "ios504" // iPhone + FB iOS app (FBIOS) API 504 — clone ios500
	PlatformIOS505 = "ios505" // iPhone + FB iOS app (FBIOS) API 505 — clone ios500
	PlatformIOS506 = "ios506" // iPhone + FB iOS app (FBIOS) API 506 — clone ios500
	PlatformIOS507 = "ios507" // iPhone + FB iOS app (FBIOS) API 507 — clone ios500
	PlatformIOS508 = "ios508" // iPhone + FB iOS app (FBIOS) API 508 — clone ios500
	PlatformIOS509 = "ios509" // iPhone + FB iOS app (FBIOS) API 509 — clone ios500
	PlatformIOS511 = "ios511" // iPhone + FB iOS app (FBIOS) API 511 — clone ios510
	PlatformIOS512 = "ios512" // iPhone + FB iOS app (FBIOS) API 512 — clone ios510
	PlatformIOS513 = "ios513" // iPhone + FB iOS app (FBIOS) API 513 — clone ios510
	PlatformIOS514 = "ios514" // iPhone + FB iOS app (FBIOS) API 514 — clone ios510
	PlatformIOS515 = "ios515" // iPhone + FB iOS app (FBIOS) API 515 — clone ios510
	PlatformIOS516 = "ios516" // iPhone + FB iOS app (FBIOS) API 516 — clone ios510
	PlatformIOS517 = "ios517" // iPhone + FB iOS app (FBIOS) API 517 — clone ios510
	PlatformIOS518 = "ios518" // iPhone + FB iOS app (FBIOS) API 518 — clone ios510
	PlatformIOS519 = "ios519" // iPhone + FB iOS app (FBIOS) API 519 — clone ios510
	PlatformIOS521 = "ios521" // iPhone + FB iOS app (FBIOS) API 521 — clone ios520
	PlatformIOS522 = "ios522" // iPhone + FB iOS app (FBIOS) API 522 — clone ios520
	PlatformIOS523 = "ios523" // iPhone + FB iOS app (FBIOS) API 523 — clone ios520
	PlatformIOS524 = "ios524" // iPhone + FB iOS app (FBIOS) API 524 — clone ios520
	PlatformIOS525 = "ios525" // iPhone + FB iOS app (FBIOS) API 525 — clone ios520
	PlatformIOS526 = "ios526" // iPhone + FB iOS app (FBIOS) API 526 — clone ios520
	PlatformIOS527 = "ios527" // iPhone + FB iOS app (FBIOS) API 527 — clone ios520
	PlatformIOS528 = "ios528" // iPhone + FB iOS app (FBIOS) API 528 — clone ios520
	PlatformIOS529 = "ios529" // iPhone + FB iOS app (FBIOS) API 529 — clone ios520
	PlatformIOS531 = "ios531" // iPhone + FB iOS app (FBIOS) API 531 — clone ios530
	PlatformIOS532 = "ios532" // iPhone + FB iOS app (FBIOS) API 532 — clone ios530
	PlatformIOS533 = "ios533" // iPhone + FB iOS app (FBIOS) API 533 — clone ios530
	PlatformIOS534 = "ios534" // iPhone + FB iOS app (FBIOS) API 534 — clone ios530
	PlatformIOS535 = "ios535" // iPhone + FB iOS app (FBIOS) API 535 — clone ios530
	PlatformIOS536 = "ios536" // iPhone + FB iOS app (FBIOS) API 536 — clone ios530
	PlatformIOS537 = "ios537" // iPhone + FB iOS app (FBIOS) API 537 — clone ios530
	PlatformIOS538 = "ios538" // iPhone + FB iOS app (FBIOS) API 538 — clone ios530
	PlatformIOS539 = "ios539" // iPhone + FB iOS app (FBIOS) API 539 — clone ios530
	PlatformIOS541 = "ios541" // iPhone + FB iOS app (FBIOS) API 541 — clone ios540
	PlatformIOS542 = "ios542" // iPhone + FB iOS app (FBIOS) API 542 — clone ios540
	PlatformIOS543 = "ios543" // iPhone + FB iOS app (FBIOS) API 543 — clone ios540
	PlatformIOS544 = "ios544" // iPhone + FB iOS app (FBIOS) API 544 — clone ios540
	PlatformIOS545 = "ios545" // iPhone + FB iOS app (FBIOS) API 545 — clone ios540
	PlatformIOS546 = "ios546" // iPhone + FB iOS app (FBIOS) API 546 — clone ios540
	PlatformIOS547 = "ios547" // iPhone + FB iOS app (FBIOS) API 547 — clone ios540
	PlatformIOS548 = "ios548" // iPhone + FB iOS app (FBIOS) API 548 — clone ios540
	PlatformIOS549 = "ios549" // iPhone + FB iOS app (FBIOS) API 549 — clone ios540
	PlatformIOS551 = "ios551" // iPhone + FB iOS app (FBIOS) API 551 — clone ios550
	PlatformIOS552 = "ios552" // iPhone + FB iOS app (FBIOS) API 552 — clone ios550
	PlatformIOS553 = "ios553" // iPhone + FB iOS app (FBIOS) API 553 — clone ios550
	PlatformIOS554 = "ios554" // iPhone + FB iOS app (FBIOS) API 554 — clone ios550
	PlatformIOS556 = "ios556" // iPhone + FB iOS app (FBIOS) API 556 — clone ios555
	PlatformIOS557 = "ios557" // iPhone + FB iOS app (FBIOS) API 557 — clone ios555
	PlatformIOS558 = "ios558" // iPhone + FB iOS app (FBIOS) API 558 — clone ios555
	PlatformIOS559 = "ios559" // iPhone + FB iOS app (FBIOS) API 559 — clone ios555
	PlatformIOS561 = "ios561" // iPhone + FB iOS app (FBIOS) API 561 — clone ios560
	PlatformIOS440 = "ios440" // iPhone + FB iOS app (FBIOS) API 440 — multi-round nosess handshake, reg+ver (clone 450)
	PlatformIOS430 = "ios430" // iPhone + FB iOS app (FBIOS) API 430 — multi-round nosess handshake, reg+ver (clone 562)
	PlatformIOS420 = "ios420" // iPhone + FB iOS app (FBIOS) API 420 — multi-round nosess handshake, reg+ver (clone 562)
	PlatformIOS560 = "ios560" // iPhone + FB iOS app (FBIOS) API 560 — multi-round nosess handshake, reg+ver (clone 555)
)

var (
	mu sync.RWMutex

	registererFactories = map[string]func() Registerer{}
	verifierFactories   = map[string]func() Verifier{}
	interactorFactories = map[string]func() Interactor{}
	feedReaderFactories = map[string]func() FeedReader{}
	securityFactories   = map[string]func() SecurityManager{}
	verifyUAFactories   = map[string]func(countryCode string) string{}
)

// ── Registration (gọi từ init() của mỗi platform package) ─────────────────

func RegisterPlatformRegisterer(platform string, factory func() Registerer) {
	mu.Lock()
	defer mu.Unlock()
	registererFactories[platform] = factory
}

func RegisterPlatformVerifier(platform string, factory func() Verifier) {
	mu.Lock()
	defer mu.Unlock()
	verifierFactories[platform] = factory
}

func RegisterPlatformInteractor(platform string, factory func() Interactor) {
	mu.Lock()
	defer mu.Unlock()
	interactorFactories[platform] = factory
}

func RegisterPlatformFeedReader(platform string, factory func() FeedReader) {
	mu.Lock()
	defer mu.Unlock()
	feedReaderFactories[platform] = factory
}

func RegisterPlatformSecurityManager(platform string, factory func() SecurityManager) {
	mu.Lock()
	defer mu.Unlock()
	securityFactories[platform] = factory
}

// RegisterPlatformVerifyUA đăng ký factory sinh User-Agent cho verify trên một platform.
// Cho phép pickUAForVerifyPlatform tạo UA đúng FBAV/FBBV của platform mà KHÔNG cần
// import package verify cụ thể (tránh circular import). Mỗi verify package gọi từ init().
//
// countryCode (ISO-2, vd "VN", "US") để factory chọn carrier + locale theo country —
// rỗng thì factory random toàn pool. Mỗi lần gọi sinh device/carrier mới (random).
func RegisterPlatformVerifyUA(platform string, factory func(countryCode string) string) {
	mu.Lock()
	defer mu.Unlock()
	verifyUAFactories[platform] = factory
}

// PlatformVerifyUA trả UA do platform sinh (random device + carrier khớp country,
// FBAV cố định theo phiên bản API). Trả "" nếu platform chưa đăng ký factory.
func PlatformVerifyUA(platform, countryCode string) string {
	mu.RLock()
	f, ok := verifyUAFactories[platform]
	mu.RUnlock()
	if !ok {
		return ""
	}
	return f(countryCode)
}

// ── Factory methods (gọi từ app.go / runner) ───────────────────────────────

// ErrUnsupportedPlatform — bản HVRIns được thiết kế cho Instagram.
// Toàn bộ API Facebook (reg/verify/interact/feed/security) đã được GỠ khỏi luồng
// chạy: cấu trúc folder các phiên bản (register/android/sXXX, register/ios/iosXXX,
// verify/...) được GIỮ NGUYÊN làm khung scaffold, nhưng dispatch tới chúng bị chặn
// cho đến khi protocol Instagram được port riêng cho từng platform.
//
// Khi port Instagram: implement Registerer/Verifier cho platform, đăng ký vào
// registry như cũ, rồi bỏ guard tương ứng ở các hàm New* bên dưới.
var ErrUnsupportedPlatform = fmt.Errorf("unsupported platform: Facebook APIs removed, Instagram implementation pending")

// NewRegisterer trả về Registerer cho nền tảng chỉ định.
// Mapping từ C#: InstanceCreateUtils.GetFacebookRegisterInstance()
// Platform package phải được blank-imported để kích hoạt registration.
func NewRegisterer(platform string) (Registerer, error) {
	// IG rebrand: mọi platform register chạy flow Instagram thật (igcore adapter),
	// thay cho logic Facebook cũ (đã gỡ). platform chỉ còn mang ý nghĩa nhãn/label.
	return newIGRegisterer(), nil
}

// NewVerifier trả về Verifier cho nền tảng chỉ định.
// Mapping từ C#: InstanceCreateUtils.GetFbVerifyAPIInstance()
func NewVerifier(platform string) (Verifier, error) {
	// Plan A (IG rebrand): chặn dispatch — không chạy logic Facebook.
	return nil, fmt.Errorf("verifier for platform %q: %w", platform, ErrUnsupportedPlatform)
}

// NewInteractor trả về Interactor cho nền tảng chỉ định.
func NewInteractor(platform string) (Interactor, error) {
	// Plan A (IG rebrand): chặn dispatch — không chạy logic Facebook.
	return nil, fmt.Errorf("interactor for platform %q: %w", platform, ErrUnsupportedPlatform)
}

// NewFeedReader trả về FeedReader cho nền tảng chỉ định.
func NewFeedReader(platform string) (FeedReader, error) {
	// Plan A (IG rebrand): chặn dispatch — không chạy logic Facebook.
	return nil, fmt.Errorf("feed reader for platform %q: %w", platform, ErrUnsupportedPlatform)
}

// NewSecurityManager trả về SecurityManager cho nền tảng chỉ định.
func NewSecurityManager(platform string) (SecurityManager, error) {
	// Plan A (IG rebrand): chặn dispatch — không chạy logic Facebook.
	return nil, fmt.Errorf("security manager for platform %q: %w", platform, ErrUnsupportedPlatform)
}
