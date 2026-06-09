package main

import (
	"context"
	"fmt"
	"strings"

	"HVRIns/internal/instagram"
	androidreg "HVRIns/internal/instagram/register/android"
	appmv3reg "HVRIns/internal/instagram/register/android/appmessv3"
	iosmessreg "HVRIns/internal/instagram/register/ios/iosmess"
	s415reg "HVRIns/internal/instagram/register/android/s415"
	s416reg "HVRIns/internal/instagram/register/android/s416"
	s417reg "HVRIns/internal/instagram/register/android/s417"
	s418reg "HVRIns/internal/instagram/register/android/s418"
	s419reg "HVRIns/internal/instagram/register/android/s419"
	s420reg "HVRIns/internal/instagram/register/android/s420"
	s421reg "HVRIns/internal/instagram/register/android/s421"
	s422reg "HVRIns/internal/instagram/register/android/s422"
	s423reg "HVRIns/internal/instagram/register/android/s423"
	s424reg "HVRIns/internal/instagram/register/android/s424"
	s425reg "HVRIns/internal/instagram/register/android/s425"
	s426reg "HVRIns/internal/instagram/register/android/s426"
	s427reg "HVRIns/internal/instagram/register/android/s427"
	s428reg "HVRIns/internal/instagram/register/android/s428"
	s429reg "HVRIns/internal/instagram/register/android/s429"
	s430reg "HVRIns/internal/instagram/register/android/s430"
	s431reg "HVRIns/internal/instagram/register/android/s431"
	s432reg "HVRIns/internal/instagram/register/android/s432"
	s433reg "HVRIns/internal/instagram/register/android/s433"
	s434reg "HVRIns/internal/instagram/register/android/s434"
	s435reg "HVRIns/internal/instagram/register/android/s435"
	s436reg "HVRIns/internal/instagram/register/android/s436"
	s437reg "HVRIns/internal/instagram/register/android/s437"
	s438reg "HVRIns/internal/instagram/register/android/s438"
	s439reg "HVRIns/internal/instagram/register/android/s439"
	s440reg "HVRIns/internal/instagram/register/android/s440"
	s441reg "HVRIns/internal/instagram/register/android/s441"
	s442reg "HVRIns/internal/instagram/register/android/s442"
	s443reg "HVRIns/internal/instagram/register/android/s443"
	s444reg "HVRIns/internal/instagram/register/android/s444"
	s445reg "HVRIns/internal/instagram/register/android/s445"
	s446reg "HVRIns/internal/instagram/register/android/s446"
	s447reg "HVRIns/internal/instagram/register/android/s447"
	s448reg "HVRIns/internal/instagram/register/android/s448"
	s449reg "HVRIns/internal/instagram/register/android/s449"
	s450reg "HVRIns/internal/instagram/register/android/s450"
	s451reg "HVRIns/internal/instagram/register/android/s451"
	s452reg "HVRIns/internal/instagram/register/android/s452"
	s453reg "HVRIns/internal/instagram/register/android/s453"
	s454reg "HVRIns/internal/instagram/register/android/s454"
	s455reg "HVRIns/internal/instagram/register/android/s455"
	s456reg "HVRIns/internal/instagram/register/android/s456"
	s457reg "HVRIns/internal/instagram/register/android/s457"
	s458reg "HVRIns/internal/instagram/register/android/s458"
	s459reg "HVRIns/internal/instagram/register/android/s459"
	s460reg "HVRIns/internal/instagram/register/android/s460"
	s461reg "HVRIns/internal/instagram/register/android/s461"
	s462reg "HVRIns/internal/instagram/register/android/s462"
	s463reg "HVRIns/internal/instagram/register/android/s463"
	s464reg "HVRIns/internal/instagram/register/android/s464"
	s465reg "HVRIns/internal/instagram/register/android/s465"
	s466reg "HVRIns/internal/instagram/register/android/s466"
	s467reg "HVRIns/internal/instagram/register/android/s467"
	s468reg "HVRIns/internal/instagram/register/android/s468"
	s469reg "HVRIns/internal/instagram/register/android/s469"
	s470reg "HVRIns/internal/instagram/register/android/s470"
	s471reg "HVRIns/internal/instagram/register/android/s471"
	s472reg "HVRIns/internal/instagram/register/android/s472"
	s473reg "HVRIns/internal/instagram/register/android/s473"
	s474reg "HVRIns/internal/instagram/register/android/s474"
	s475reg "HVRIns/internal/instagram/register/android/s475"
	s476reg "HVRIns/internal/instagram/register/android/s476"
	s477reg "HVRIns/internal/instagram/register/android/s477"
	s478reg "HVRIns/internal/instagram/register/android/s478"
	s479reg "HVRIns/internal/instagram/register/android/s479"
	s480reg "HVRIns/internal/instagram/register/android/s480"
	s481reg "HVRIns/internal/instagram/register/android/s481"
	s482reg "HVRIns/internal/instagram/register/android/s482"
	s483reg "HVRIns/internal/instagram/register/android/s483"
	s484reg "HVRIns/internal/instagram/register/android/s484"
	s485reg "HVRIns/internal/instagram/register/android/s485"
	s486reg "HVRIns/internal/instagram/register/android/s486"
	s487reg "HVRIns/internal/instagram/register/android/s487"
	s488reg "HVRIns/internal/instagram/register/android/s488"
	s489reg "HVRIns/internal/instagram/register/android/s489"
	s490reg "HVRIns/internal/instagram/register/android/s490"
	s491reg "HVRIns/internal/instagram/register/android/s491"
	s492reg "HVRIns/internal/instagram/register/android/s492"
	s493reg "HVRIns/internal/instagram/register/android/s493"
	s494reg "HVRIns/internal/instagram/register/android/s494"
	s495reg "HVRIns/internal/instagram/register/android/s495"
	s496reg "HVRIns/internal/instagram/register/android/s496"
	s497reg "HVRIns/internal/instagram/register/android/s497"
	s498reg "HVRIns/internal/instagram/register/android/s498"
	s499reg "HVRIns/internal/instagram/register/android/s499"
	s500reg "HVRIns/internal/instagram/register/android/s500"
	s501reg "HVRIns/internal/instagram/register/android/s501"
	s502reg "HVRIns/internal/instagram/register/android/s502"
	s503reg "HVRIns/internal/instagram/register/android/s503"
	s504reg "HVRIns/internal/instagram/register/android/s504"
	s505reg "HVRIns/internal/instagram/register/android/s505"
	s506reg "HVRIns/internal/instagram/register/android/s506"
	s507reg "HVRIns/internal/instagram/register/android/s507"
	s508reg "HVRIns/internal/instagram/register/android/s508"
	s509reg "HVRIns/internal/instagram/register/android/s509"
	s510reg "HVRIns/internal/instagram/register/android/s510"
	s511reg "HVRIns/internal/instagram/register/android/s511"
	s512reg "HVRIns/internal/instagram/register/android/s512"
	s513reg "HVRIns/internal/instagram/register/android/s513"
	s514reg "HVRIns/internal/instagram/register/android/s514"
	s515reg "HVRIns/internal/instagram/register/android/s515"
	s516reg "HVRIns/internal/instagram/register/android/s516"
	s517reg "HVRIns/internal/instagram/register/android/s517"
	s518reg "HVRIns/internal/instagram/register/android/s518"
	s519reg "HVRIns/internal/instagram/register/android/s519"
	s520reg "HVRIns/internal/instagram/register/android/s520"
	s521reg "HVRIns/internal/instagram/register/android/s521"
	s522reg "HVRIns/internal/instagram/register/android/s522"
	s523reg "HVRIns/internal/instagram/register/android/s523"
	s524reg "HVRIns/internal/instagram/register/android/s524"
	s525reg "HVRIns/internal/instagram/register/android/s525"
	s526reg "HVRIns/internal/instagram/register/android/s526"
	s527reg "HVRIns/internal/instagram/register/android/s527"
	s528reg "HVRIns/internal/instagram/register/android/s528"
	s529reg "HVRIns/internal/instagram/register/android/s529"
	s530reg "HVRIns/internal/instagram/register/android/s530"
	s531reg "HVRIns/internal/instagram/register/android/s531"
	s532reg "HVRIns/internal/instagram/register/android/s532"
	s533reg "HVRIns/internal/instagram/register/android/s533"
	s534reg "HVRIns/internal/instagram/register/android/s534"
	s535reg "HVRIns/internal/instagram/register/android/s535"
	s536reg "HVRIns/internal/instagram/register/android/s536"
	s537reg "HVRIns/internal/instagram/register/android/s537"
	s538reg "HVRIns/internal/instagram/register/android/s538"
	s539reg "HVRIns/internal/instagram/register/android/s539"
	s540reg "HVRIns/internal/instagram/register/android/s540"
	s541reg "HVRIns/internal/instagram/register/android/s541"
	s542reg "HVRIns/internal/instagram/register/android/s542"
	s543reg "HVRIns/internal/instagram/register/android/s543"
	s544reg "HVRIns/internal/instagram/register/android/s544"
	s545reg "HVRIns/internal/instagram/register/android/s545"
	s546reg "HVRIns/internal/instagram/register/android/s546"
	s547reg "HVRIns/internal/instagram/register/android/s547"
	s548reg "HVRIns/internal/instagram/register/android/s548"
	s549reg "HVRIns/internal/instagram/register/android/s549"
	s550reg "HVRIns/internal/instagram/register/android/s550"
	s551reg "HVRIns/internal/instagram/register/android/s551"
	s552reg "HVRIns/internal/instagram/register/android/s552"
	s553reg "HVRIns/internal/instagram/register/android/s553"
	s554reg "HVRIns/internal/instagram/register/android/s554"
	s555reg "HVRIns/internal/instagram/register/android/s555"
	s555v2reg "HVRIns/internal/instagram/register/android/s555v2"
	s556reg "HVRIns/internal/instagram/register/android/s556"
	s557reg "HVRIns/internal/instagram/register/android/s557"
	s558reg "HVRIns/internal/instagram/register/android/s558"
	s558v2reg "HVRIns/internal/instagram/register/android/s558v2"
	s559reg "HVRIns/internal/instagram/register/android/s559"
	s559v2reg "HVRIns/internal/instagram/register/android/s559v2"
	s560reg "HVRIns/internal/instagram/register/android/s560"
	s560v2reg "HVRIns/internal/instagram/register/android/s560v2"
	s561reg "HVRIns/internal/instagram/register/android/s561"
	s561v2reg "HVRIns/internal/instagram/register/android/s561v2"
	s561v3reg "HVRIns/internal/instagram/register/android/s561v3"
	s561v4s21reg "HVRIns/internal/instagram/register/android/s561v4s21"
	s561v4s23reg "HVRIns/internal/instagram/register/android/s561v4s23"
	s561v99reg "HVRIns/internal/instagram/register/android/s561v99"
	s562reg "HVRIns/internal/instagram/register/android/s562"
	s562v3reg "HVRIns/internal/instagram/register/android/s562v3"
	s562v4s21reg "HVRIns/internal/instagram/register/android/s562v4s21"
	s562v4s23reg "HVRIns/internal/instagram/register/android/s562v4s23"
	s563reg "HVRIns/internal/instagram/register/android/s563"
	s563s21reg "HVRIns/internal/instagram/register/android/s563s21"
	s563v3s21reg "HVRIns/internal/instagram/register/android/s563v3s21"
	s563v4s21reg "HVRIns/internal/instagram/register/android/s563v4s21"
	s563v4s23reg "HVRIns/internal/instagram/register/android/s563v4s23"
	s563v5s21reg "HVRIns/internal/instagram/register/android/s563v5s21"
	s563v5s23reg "HVRIns/internal/instagram/register/android/s563v5s23"
	s563v6s21reg "HVRIns/internal/instagram/register/android/s563v6s21"
	s563v6s23reg "HVRIns/internal/instagram/register/android/s563v6s23"
	s564v1s21reg "HVRIns/internal/instagram/register/android/s564v1s21"
	s564v1s23reg "HVRIns/internal/instagram/register/android/s564v1s23"
	s564v2s21reg "HVRIns/internal/instagram/register/android/s564v2s21"
	s564v2s23reg "HVRIns/internal/instagram/register/android/s564v2s23"
	s564v3s21reg "HVRIns/internal/instagram/register/android/s564v3s21"
	s564v3s23reg "HVRIns/internal/instagram/register/android/s564v3s23"
	s565s21reg "HVRIns/internal/instagram/register/android/s565s21"
	s565s23reg "HVRIns/internal/instagram/register/android/s565s23"
	s565v2s21reg "HVRIns/internal/instagram/register/android/s565v2s21"
	s565v2s23reg "HVRIns/internal/instagram/register/android/s565v2s23"

	_ "HVRIns/internal/instagram/register/ios/ios562"
)

// regSxxxWorkerContext — interface chung cho tất cả platform S4xx/S5xx (S415..S562).
// Thêm platform mới: implement interface này + thêm vào 3 func bên dưới.
type regSxxxWorkerContext interface {
	Close()
	SetLocale(string)
	SetConnectionType(string)
	SetUAOptions(bool)
	SetUA(string)
	UserAgent() string
	Register(context.Context, *instagram.RegInput, func(string)) *instagram.RegResult
}

// regPlatformList trả về danh sách platform reg user đã chọn (hỗ trợ multi-version).
//   - ApiRegPlatforms (len>0) → dùng list này, trim + bỏ rỗng + dedup, giữ thứ tự.
//   - Rỗng → fallback [ApiRegPlatform].
//   - Vẫn rỗng → [PlatformWeb] (giống default cũ).
func regPlatformList(c InteractionConfig) []string {
	seen := make(map[string]struct{})
	out := make([]string, 0, len(c.ApiRegPlatforms)+1)
	add := func(p string) {
		p = strings.TrimSpace(p)
		if p == "" {
			return
		}
		key := strings.ToLower(p)
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		out = append(out, p)
	}
	for _, p := range c.ApiRegPlatforms {
		add(p)
	}
	if len(out) == 0 {
		add(c.ApiRegPlatform)
	}
	if len(out) == 0 {
		out = append(out, instagram.PlatformWeb)
	}
	return out
}

func isRegPlatformSxxx(platform string) bool {
	p := strings.ToLower(strings.TrimSpace(platform))
	switch p {
	case instagram.PlatformS415, instagram.PlatformS425, instagram.PlatformS435, instagram.PlatformS445,
		instagram.PlatformS416, instagram.PlatformS417, instagram.PlatformS418, instagram.PlatformS419, instagram.PlatformS420, instagram.PlatformS421,
		instagram.PlatformS422, instagram.PlatformS423, instagram.PlatformS424, instagram.PlatformS426, instagram.PlatformS427, instagram.PlatformS428,
		instagram.PlatformS429, instagram.PlatformS430, instagram.PlatformS431, instagram.PlatformS432, instagram.PlatformS433, instagram.PlatformS434,
		instagram.PlatformS436, instagram.PlatformS437, instagram.PlatformS438, instagram.PlatformS439, instagram.PlatformS440, instagram.PlatformS441,
		instagram.PlatformS442, instagram.PlatformS443, instagram.PlatformS444,
		instagram.PlatformS446, instagram.PlatformS447, instagram.PlatformS448, instagram.PlatformS449, instagram.PlatformS450, instagram.PlatformS451,
		instagram.PlatformS452, instagram.PlatformS453, instagram.PlatformS454, instagram.PlatformS455,
		instagram.PlatformS456, instagram.PlatformS457, instagram.PlatformS458, instagram.PlatformS459, instagram.PlatformS460,
		instagram.PlatformS461, instagram.PlatformS462, instagram.PlatformS463, instagram.PlatformS464, instagram.PlatformS465,
		instagram.PlatformS466, instagram.PlatformS467, instagram.PlatformS468, instagram.PlatformS469, instagram.PlatformS470,
		instagram.PlatformS471, instagram.PlatformS472, instagram.PlatformS473, instagram.PlatformS474, instagram.PlatformS475,
		instagram.PlatformS476, instagram.PlatformS477, instagram.PlatformS478, instagram.PlatformS479, instagram.PlatformS480,
		instagram.PlatformS481, instagram.PlatformS482, instagram.PlatformS483, instagram.PlatformS484, instagram.PlatformS485,
		instagram.PlatformS486, instagram.PlatformS487, instagram.PlatformS488, instagram.PlatformS489, instagram.PlatformS490,
		instagram.PlatformS491, instagram.PlatformS492, instagram.PlatformS493, instagram.PlatformS494, instagram.PlatformS496,
		instagram.PlatformS497, instagram.PlatformS498, instagram.PlatformS499,
		instagram.PlatformS495,
		instagram.PlatformS500, instagram.PlatformS501, instagram.PlatformS502, instagram.PlatformS503, instagram.PlatformS504,
		instagram.PlatformS505, instagram.PlatformS506, instagram.PlatformS507, instagram.PlatformS508, instagram.PlatformS509,
		instagram.PlatformS510, instagram.PlatformS511, instagram.PlatformS512, instagram.PlatformS513, instagram.PlatformS514,
		instagram.PlatformS515, instagram.PlatformS516, instagram.PlatformS517, instagram.PlatformS518, instagram.PlatformS519,
		instagram.PlatformS520, instagram.PlatformS521, instagram.PlatformS522, instagram.PlatformS523, instagram.PlatformS524,
		instagram.PlatformS525, instagram.PlatformS526, instagram.PlatformS527, instagram.PlatformS528, instagram.PlatformS529,
		instagram.PlatformS530, instagram.PlatformS531, instagram.PlatformS532,
		instagram.PlatformS533,
		instagram.PlatformS534, instagram.PlatformS535, instagram.PlatformS536, instagram.PlatformS537, instagram.PlatformS538,
		instagram.PlatformS539, instagram.PlatformS540, instagram.PlatformS541, instagram.PlatformS542, instagram.PlatformS543,
		instagram.PlatformS544,
		instagram.PlatformS545, instagram.PlatformS546, instagram.PlatformS547, instagram.PlatformS548, instagram.PlatformS549,
		instagram.PlatformS550, instagram.PlatformS551, instagram.PlatformS552, instagram.PlatformS553, instagram.PlatformS554,
		instagram.PlatformS555, instagram.PlatformS556, instagram.PlatformS557, instagram.PlatformS558,
		instagram.PlatformS555V2,
		instagram.PlatformS558V2,
		instagram.PlatformS559, instagram.PlatformS559V2,
		instagram.PlatformS560, instagram.PlatformS560V2,
		instagram.PlatformS561, instagram.PlatformS561V2, instagram.PlatformS562,
		instagram.PlatformS561V3, instagram.PlatformS561V99,
		instagram.PlatformS561V4S21, instagram.PlatformS561V4S23,
		instagram.PlatformS562V3, instagram.PlatformS562V4S21, instagram.PlatformS562V4S23,
		instagram.PlatformS563, instagram.PlatformS563S21, instagram.PlatformS563V3S21, instagram.PlatformS563V4S21, instagram.PlatformS563V4S23, instagram.PlatformS563V5S21, instagram.PlatformS563V5S23,
		instagram.PlatformS563V6S21, instagram.PlatformS563V6S23,
		instagram.PlatformS564V1S21, instagram.PlatformS564V1S23, instagram.PlatformS564V2S21, instagram.PlatformS564V2S23,
		instagram.PlatformS564V3S21, instagram.PlatformS564V3S23,
		instagram.PlatformS565S21, instagram.PlatformS565S23,
		instagram.PlatformS565V2S21, instagram.PlatformS565V2S23,
		instagram.PlatformAppMV3, instagram.PlatformAppMV3_535, instagram.PlatformAppMV3_545,
		instagram.PlatformAppMV3_555, instagram.PlatformAppMV3_563, instagram.PlatformAppMV3_564,
		instagram.PlatformAppMV3_565, instagram.PlatformAppMV3_525, instagram.PlatformAppMV3_515, instagram.PlatformAppMV3_505,
		instagram.PlatformAppMV3_490,
		instagram.PlatformIOSMessReg:
		return true
	default:
		return false
	}
}

func regPoolForSxxx(platform string) *androidreg.PartitionedDatrPool {
	switch strings.ToLower(strings.TrimSpace(platform)) {
	case instagram.PlatformS415:
		return s415reg.SharedPool
	case instagram.PlatformS425:
		return s425reg.SharedPool
	case instagram.PlatformS435:
		return s435reg.SharedPool
	case instagram.PlatformS445:
		return s445reg.SharedPool
	case instagram.PlatformS416:
		return s416reg.SharedPool
	case instagram.PlatformS417:
		return s417reg.SharedPool
	case instagram.PlatformS418:
		return s418reg.SharedPool
	case instagram.PlatformS419:
		return s419reg.SharedPool
	case instagram.PlatformS420:
		return s420reg.SharedPool
	case instagram.PlatformS421:
		return s421reg.SharedPool
	case instagram.PlatformS422:
		return s422reg.SharedPool
	case instagram.PlatformS423:
		return s423reg.SharedPool
	case instagram.PlatformS424:
		return s424reg.SharedPool
	case instagram.PlatformS426:
		return s426reg.SharedPool
	case instagram.PlatformS427:
		return s427reg.SharedPool
	case instagram.PlatformS428:
		return s428reg.SharedPool
	case instagram.PlatformS429:
		return s429reg.SharedPool
	case instagram.PlatformS430:
		return s430reg.SharedPool
	case instagram.PlatformS431:
		return s431reg.SharedPool
	case instagram.PlatformS432:
		return s432reg.SharedPool
	case instagram.PlatformS433:
		return s433reg.SharedPool
	case instagram.PlatformS434:
		return s434reg.SharedPool
	case instagram.PlatformS436:
		return s436reg.SharedPool
	case instagram.PlatformS437:
		return s437reg.SharedPool
	case instagram.PlatformS438:
		return s438reg.SharedPool
	case instagram.PlatformS439:
		return s439reg.SharedPool
	case instagram.PlatformS440:
		return s440reg.SharedPool
	case instagram.PlatformS441:
		return s441reg.SharedPool
	case instagram.PlatformS442:
		return s442reg.SharedPool
	case instagram.PlatformS443:
		return s443reg.SharedPool
	case instagram.PlatformS444:
		return s444reg.SharedPool
	case instagram.PlatformS446:
		return s446reg.SharedPool
	case instagram.PlatformS447:
		return s447reg.SharedPool
	case instagram.PlatformS448:
		return s448reg.SharedPool
	case instagram.PlatformS449:
		return s449reg.SharedPool
	case instagram.PlatformS450:
		return s450reg.SharedPool
	case instagram.PlatformS451:
		return s451reg.SharedPool
	case instagram.PlatformS452:
		return s452reg.SharedPool
	case instagram.PlatformS453:
		return s453reg.SharedPool
	case instagram.PlatformS454:
		return s454reg.SharedPool
	case instagram.PlatformS455:
		return s455reg.SharedPool
	case instagram.PlatformS456:
		return s456reg.SharedPool
	case instagram.PlatformS457:
		return s457reg.SharedPool
	case instagram.PlatformS458:
		return s458reg.SharedPool
	case instagram.PlatformS459:
		return s459reg.SharedPool
	case instagram.PlatformS460:
		return s460reg.SharedPool
	case instagram.PlatformS461:
		return s461reg.SharedPool
	case instagram.PlatformS462:
		return s462reg.SharedPool
	case instagram.PlatformS463:
		return s463reg.SharedPool
	case instagram.PlatformS464:
		return s464reg.SharedPool
	case instagram.PlatformS465:
		return s465reg.SharedPool
	case instagram.PlatformS466:
		return s466reg.SharedPool
	case instagram.PlatformS467:
		return s467reg.SharedPool
	case instagram.PlatformS468:
		return s468reg.SharedPool
	case instagram.PlatformS469:
		return s469reg.SharedPool
	case instagram.PlatformS470:
		return s470reg.SharedPool
	case instagram.PlatformS471:
		return s471reg.SharedPool
	case instagram.PlatformS472:
		return s472reg.SharedPool
	case instagram.PlatformS473:
		return s473reg.SharedPool
	case instagram.PlatformS474:
		return s474reg.SharedPool
	case instagram.PlatformS475:
		return s475reg.SharedPool
	case instagram.PlatformS476:
		return s476reg.SharedPool
	case instagram.PlatformS477:
		return s477reg.SharedPool
	case instagram.PlatformS478:
		return s478reg.SharedPool
	case instagram.PlatformS479:
		return s479reg.SharedPool
	case instagram.PlatformS480:
		return s480reg.SharedPool
	case instagram.PlatformS481:
		return s481reg.SharedPool
	case instagram.PlatformS482:
		return s482reg.SharedPool
	case instagram.PlatformS483:
		return s483reg.SharedPool
	case instagram.PlatformS484:
		return s484reg.SharedPool
	case instagram.PlatformS485:
		return s485reg.SharedPool
	case instagram.PlatformS486:
		return s486reg.SharedPool
	case instagram.PlatformS487:
		return s487reg.SharedPool
	case instagram.PlatformS488:
		return s488reg.SharedPool
	case instagram.PlatformS489:
		return s489reg.SharedPool
	case instagram.PlatformS490:
		return s490reg.SharedPool
	case instagram.PlatformS491:
		return s491reg.SharedPool
	case instagram.PlatformS492:
		return s492reg.SharedPool
	case instagram.PlatformS493:
		return s493reg.SharedPool
	case instagram.PlatformS494:
		return s494reg.SharedPool
	case instagram.PlatformS496:
		return s496reg.SharedPool
	case instagram.PlatformS497:
		return s497reg.SharedPool
	case instagram.PlatformS498:
		return s498reg.SharedPool
	case instagram.PlatformS499:
		return s499reg.SharedPool
	case instagram.PlatformS495:
		return s495reg.SharedPool
	case instagram.PlatformS500:
		return s500reg.SharedPool
	case instagram.PlatformS501:
		return s501reg.SharedPool
	case instagram.PlatformS502:
		return s502reg.SharedPool
	case instagram.PlatformS503:
		return s503reg.SharedPool
	case instagram.PlatformS504:
		return s504reg.SharedPool
	case instagram.PlatformS505:
		return s505reg.SharedPool
	case instagram.PlatformS506:
		return s506reg.SharedPool
	case instagram.PlatformS507:
		return s507reg.SharedPool
	case instagram.PlatformS508:
		return s508reg.SharedPool
	case instagram.PlatformS509:
		return s509reg.SharedPool
	case instagram.PlatformS510:
		return s510reg.SharedPool
	case instagram.PlatformS511:
		return s511reg.SharedPool
	case instagram.PlatformS512:
		return s512reg.SharedPool
	case instagram.PlatformS513:
		return s513reg.SharedPool
	case instagram.PlatformS514:
		return s514reg.SharedPool
	case instagram.PlatformS515:
		return s515reg.SharedPool
	case instagram.PlatformS516:
		return s516reg.SharedPool
	case instagram.PlatformS517:
		return s517reg.SharedPool
	case instagram.PlatformS518:
		return s518reg.SharedPool
	case instagram.PlatformS519:
		return s519reg.SharedPool
	case instagram.PlatformS520:
		return s520reg.SharedPool
	case instagram.PlatformS521:
		return s521reg.SharedPool
	case instagram.PlatformS522:
		return s522reg.SharedPool
	case instagram.PlatformS523:
		return s523reg.SharedPool
	case instagram.PlatformS524:
		return s524reg.SharedPool
	case instagram.PlatformS525:
		return s525reg.SharedPool
	case instagram.PlatformS526:
		return s526reg.SharedPool
	case instagram.PlatformS527:
		return s527reg.SharedPool
	case instagram.PlatformS528:
		return s528reg.SharedPool
	case instagram.PlatformS529:
		return s529reg.SharedPool
	case instagram.PlatformS530:
		return s530reg.SharedPool
	case instagram.PlatformS531:
		return s531reg.SharedPool
	case instagram.PlatformS532:
		return s532reg.SharedPool
	case instagram.PlatformS533:
		return s533reg.SharedPool
	case instagram.PlatformS534:
		return s534reg.SharedPool
	case instagram.PlatformS535:
		return s535reg.SharedPool
	case instagram.PlatformS536:
		return s536reg.SharedPool
	case instagram.PlatformS537:
		return s537reg.SharedPool
	case instagram.PlatformS538:
		return s538reg.SharedPool
	case instagram.PlatformS539:
		return s539reg.SharedPool
	case instagram.PlatformS540:
		return s540reg.SharedPool
	case instagram.PlatformS541:
		return s541reg.SharedPool
	case instagram.PlatformS542:
		return s542reg.SharedPool
	case instagram.PlatformS543:
		return s543reg.SharedPool
	case instagram.PlatformS544:
		return s544reg.SharedPool
	case instagram.PlatformS545:
		return s545reg.SharedPool
	case instagram.PlatformS546:
		return s546reg.SharedPool
	case instagram.PlatformS547:
		return s547reg.SharedPool
	case instagram.PlatformS548:
		return s548reg.SharedPool
	case instagram.PlatformS549:
		return s549reg.SharedPool
	case instagram.PlatformS550:
		return s550reg.SharedPool
	case instagram.PlatformS551:
		return s551reg.SharedPool
	case instagram.PlatformS552:
		return s552reg.SharedPool
	case instagram.PlatformS553:
		return s553reg.SharedPool
	case instagram.PlatformS554:
		return s554reg.SharedPool
	case instagram.PlatformS555:
		return s555reg.SharedPool
	case instagram.PlatformS555V2:
		return s555v2reg.SharedPool
	case instagram.PlatformS556:
		return s556reg.SharedPool
	case instagram.PlatformS557:
		return s557reg.SharedPool
	case instagram.PlatformS558:
		return s558reg.SharedPool
	case instagram.PlatformS558V2:
		return s558v2reg.SharedPool
	case instagram.PlatformS559:
		return s559reg.SharedPool
	case instagram.PlatformS559V2:
		return s559v2reg.SharedPool
	case instagram.PlatformS560:
		return s560reg.SharedPool
	case instagram.PlatformS560V2:
		return s560v2reg.SharedPool
	case instagram.PlatformS561:
		return s561reg.SharedPool
	case instagram.PlatformS561V2:
		return s561v2reg.SharedPool
	case instagram.PlatformS561V3:
		return s561v3reg.SharedPool
	case instagram.PlatformS561V99:
		return s561v99reg.SharedPool
	case instagram.PlatformS561V4S21:
		return s561v4s21reg.SharedPool
	case instagram.PlatformS561V4S23:
		return s561v4s23reg.SharedPool
	case instagram.PlatformS562:
		return s562reg.SharedPool
	case instagram.PlatformS562V3:
		return s562v3reg.SharedPool
	case instagram.PlatformS562V4S21:
		return s562v4s21reg.SharedPool
	case instagram.PlatformS562V4S23:
		return s562v4s23reg.SharedPool
	case instagram.PlatformS563:
		return s563reg.SharedPool
	case instagram.PlatformS563S21:
		return s563s21reg.SharedPool
	case instagram.PlatformS563V3S21:
		return s563v3s21reg.SharedPool
	case instagram.PlatformS563V4S21:
		return s563v4s21reg.SharedPool
	case instagram.PlatformS563V4S23:
		return s563v4s23reg.SharedPool
	case instagram.PlatformS563V5S21:
		return s563v5s21reg.SharedPool
	case instagram.PlatformS563V5S23:
		return s563v5s23reg.SharedPool
	case instagram.PlatformS563V6S21:
		return s563v6s21reg.SharedPool
	case instagram.PlatformS563V6S23:
		return s563v6s23reg.SharedPool
	case instagram.PlatformS564V1S21:
		return s564v1s21reg.SharedPool
	case instagram.PlatformS564V1S23:
		return s564v1s23reg.SharedPool
	case instagram.PlatformS564V2S21:
		return s564v2s21reg.SharedPool
	case instagram.PlatformS564V2S23:
		return s564v2s23reg.SharedPool
	case instagram.PlatformS564V3S21:
		return s564v3s21reg.SharedPool
	case instagram.PlatformS564V3S23:
		return s564v3s23reg.SharedPool
	case instagram.PlatformS565S21:
		return s565s21reg.SharedPool
	case instagram.PlatformS565S23:
		return s565s23reg.SharedPool
	case instagram.PlatformS565V2S21:
		return s565v2s21reg.SharedPool
	case instagram.PlatformS565V2S23:
		return s565v2s23reg.SharedPool
	case instagram.PlatformAppMV3, instagram.PlatformAppMV3_535, instagram.PlatformAppMV3_545,
		instagram.PlatformAppMV3_555, instagram.PlatformAppMV3_563, instagram.PlatformAppMV3_564,
		instagram.PlatformAppMV3_565, instagram.PlatformAppMV3_525, instagram.PlatformAppMV3_515, instagram.PlatformAppMV3_505, instagram.PlatformAppMV3_490:
		return appmv3reg.SharedPool
	case instagram.PlatformIOSMessReg:
		return iosmessreg.SharedPool
	default:
		return nil
	}
}

func regSxxxPoolPointers() map[string]**androidreg.PartitionedDatrPool {
	return map[string]**androidreg.PartitionedDatrPool{
		"S415":      &s415reg.SharedPool,
		"S425":      &s425reg.SharedPool,
		"S435":      &s435reg.SharedPool,
		"S445":      &s445reg.SharedPool,
		"S416":      &s416reg.SharedPool,
		"S417":      &s417reg.SharedPool,
		"S418":      &s418reg.SharedPool,
		"S419":      &s419reg.SharedPool,
		"S420":      &s420reg.SharedPool,
		"S421":      &s421reg.SharedPool,
		"S422":      &s422reg.SharedPool,
		"S423":      &s423reg.SharedPool,
		"S424":      &s424reg.SharedPool,
		"S426":      &s426reg.SharedPool,
		"S427":      &s427reg.SharedPool,
		"S428":      &s428reg.SharedPool,
		"S429":      &s429reg.SharedPool,
		"S430":      &s430reg.SharedPool,
		"S431":      &s431reg.SharedPool,
		"S432":      &s432reg.SharedPool,
		"S433":      &s433reg.SharedPool,
		"S434":      &s434reg.SharedPool,
		"S436":      &s436reg.SharedPool,
		"S437":      &s437reg.SharedPool,
		"S438":      &s438reg.SharedPool,
		"S439":      &s439reg.SharedPool,
		"S440":      &s440reg.SharedPool,
		"S441":      &s441reg.SharedPool,
		"S442":      &s442reg.SharedPool,
		"S443":      &s443reg.SharedPool,
		"S444":      &s444reg.SharedPool,
		"S446":      &s446reg.SharedPool,
		"S447":      &s447reg.SharedPool,
		"S448":      &s448reg.SharedPool,
		"S449":      &s449reg.SharedPool,
		"S450":      &s450reg.SharedPool,
		"S451":      &s451reg.SharedPool,
		"S452":      &s452reg.SharedPool,
		"S453":      &s453reg.SharedPool,
		"S454":      &s454reg.SharedPool,
		"S455":      &s455reg.SharedPool,
		"S456":      &s456reg.SharedPool,
		"S457":      &s457reg.SharedPool,
		"S458":      &s458reg.SharedPool,
		"S459":      &s459reg.SharedPool,
		"S460":      &s460reg.SharedPool,
		"S461":      &s461reg.SharedPool,
		"S462":      &s462reg.SharedPool,
		"S463":      &s463reg.SharedPool,
		"S464":      &s464reg.SharedPool,
		"S465":      &s465reg.SharedPool,
		"S466":      &s466reg.SharedPool,
		"S467":      &s467reg.SharedPool,
		"S468":      &s468reg.SharedPool,
		"S469":      &s469reg.SharedPool,
		"S470":      &s470reg.SharedPool,
		"S471":      &s471reg.SharedPool,
		"S472":      &s472reg.SharedPool,
		"S473":      &s473reg.SharedPool,
		"S474":      &s474reg.SharedPool,
		"S475":      &s475reg.SharedPool,
		"S476":      &s476reg.SharedPool,
		"S477":      &s477reg.SharedPool,
		"S478":      &s478reg.SharedPool,
		"S479":      &s479reg.SharedPool,
		"S480":      &s480reg.SharedPool,
		"S481":      &s481reg.SharedPool,
		"S482":      &s482reg.SharedPool,
		"S483":      &s483reg.SharedPool,
		"S484":      &s484reg.SharedPool,
		"S485":      &s485reg.SharedPool,
		"S486":      &s486reg.SharedPool,
		"S487":      &s487reg.SharedPool,
		"S488":      &s488reg.SharedPool,
		"S489":      &s489reg.SharedPool,
		"S490":      &s490reg.SharedPool,
		"S491":      &s491reg.SharedPool,
		"S492":      &s492reg.SharedPool,
		"S493":      &s493reg.SharedPool,
		"S494":      &s494reg.SharedPool,
		"S496":      &s496reg.SharedPool,
		"S497":      &s497reg.SharedPool,
		"S498":      &s498reg.SharedPool,
		"S499":      &s499reg.SharedPool,
		"S495":      &s495reg.SharedPool,
		"S500":      &s500reg.SharedPool,
		"S501":      &s501reg.SharedPool,
		"S502":      &s502reg.SharedPool,
		"S503":      &s503reg.SharedPool,
		"S504":      &s504reg.SharedPool,
		"S505":      &s505reg.SharedPool,
		"S506":      &s506reg.SharedPool,
		"S507":      &s507reg.SharedPool,
		"S508":      &s508reg.SharedPool,
		"S509":      &s509reg.SharedPool,
		"S510":      &s510reg.SharedPool,
		"S511":      &s511reg.SharedPool,
		"S512":      &s512reg.SharedPool,
		"S513":      &s513reg.SharedPool,
		"S514":      &s514reg.SharedPool,
		"S515":      &s515reg.SharedPool,
		"S516":      &s516reg.SharedPool,
		"S517":      &s517reg.SharedPool,
		"S518":      &s518reg.SharedPool,
		"S519":      &s519reg.SharedPool,
		"S520":      &s520reg.SharedPool,
		"S521":      &s521reg.SharedPool,
		"S522":      &s522reg.SharedPool,
		"S523":      &s523reg.SharedPool,
		"S524":      &s524reg.SharedPool,
		"S525":      &s525reg.SharedPool,
		"S526":      &s526reg.SharedPool,
		"S527":      &s527reg.SharedPool,
		"S528":      &s528reg.SharedPool,
		"S529":      &s529reg.SharedPool,
		"S530":      &s530reg.SharedPool,
		"S531":      &s531reg.SharedPool,
		"S532":      &s532reg.SharedPool,
		"S533":      &s533reg.SharedPool,
		"S534":      &s534reg.SharedPool,
		"S535":      &s535reg.SharedPool,
		"S536":      &s536reg.SharedPool,
		"S537":      &s537reg.SharedPool,
		"S538":      &s538reg.SharedPool,
		"S539":      &s539reg.SharedPool,
		"S540":      &s540reg.SharedPool,
		"S541":      &s541reg.SharedPool,
		"S542":      &s542reg.SharedPool,
		"S543":      &s543reg.SharedPool,
		"S544":      &s544reg.SharedPool,
		"S545":      &s545reg.SharedPool,
		"S546":      &s546reg.SharedPool,
		"S547":      &s547reg.SharedPool,
		"S548":      &s548reg.SharedPool,
		"S549":      &s549reg.SharedPool,
		"S550":      &s550reg.SharedPool,
		"S551":      &s551reg.SharedPool,
		"S552":      &s552reg.SharedPool,
		"S553":      &s553reg.SharedPool,
		"S554":      &s554reg.SharedPool,
		"S555":      &s555reg.SharedPool,
		"S555V2":    &s555v2reg.SharedPool,
		"S556":      &s556reg.SharedPool,
		"S557":      &s557reg.SharedPool,
		"S558":      &s558reg.SharedPool,
		"S558V2":    &s558v2reg.SharedPool,
		"S559":      &s559reg.SharedPool,
		"S559V2":    &s559v2reg.SharedPool,
		"S560":      &s560reg.SharedPool,
		"S560V2":    &s560v2reg.SharedPool,
		"S561":      &s561reg.SharedPool,
		"S561V2":    &s561v2reg.SharedPool,
		"S561V3":    &s561v3reg.SharedPool,
		"S561V99":   &s561v99reg.SharedPool,
		"S561V4S21": &s561v4s21reg.SharedPool,
		"S561V4S23": &s561v4s23reg.SharedPool,
		"S562":      &s562reg.SharedPool,
		"S562V3":    &s562v3reg.SharedPool,
		"S562V4S21": &s562v4s21reg.SharedPool,
		"S562V4S23": &s562v4s23reg.SharedPool,
		"S563":      &s563reg.SharedPool,
		"S563S21":   &s563s21reg.SharedPool,
		"S563V3S21": &s563v3s21reg.SharedPool,
		"S563V4S21": &s563v4s21reg.SharedPool,
		"S563V4S23": &s563v4s23reg.SharedPool,
		"S563V5S21": &s563v5s21reg.SharedPool,
		"S563V5S23": &s563v5s23reg.SharedPool,
		"S563V6S21": &s563v6s21reg.SharedPool,
		"S563V6S23": &s563v6s23reg.SharedPool,
		"S564V1S21": &s564v1s21reg.SharedPool,
		"S564V1S23": &s564v1s23reg.SharedPool,
		"S564V2S21": &s564v2s21reg.SharedPool,
		"S564V2S23": &s564v2s23reg.SharedPool,
		"S564V3S21": &s564v3s21reg.SharedPool,
		"S564V3S23": &s564v3s23reg.SharedPool,
		"S565":      &s565s21reg.SharedPool,
		"S565S23":   &s565s23reg.SharedPool,
		"S565V2":    &s565v2s21reg.SharedPool,
		"S565V2S23": &s565v2s23reg.SharedPool,
		"APPMV3":    &appmv3reg.SharedPool,
		"APPMV3535": &appmv3reg.SharedPool,
		"APPMV3545": &appmv3reg.SharedPool,
		"APPMV3555": &appmv3reg.SharedPool,
		"APPMV3563": &appmv3reg.SharedPool,
		"APPMV3564": &appmv3reg.SharedPool,
		"APPMV3565": &appmv3reg.SharedPool,
		"APPMV3525": &appmv3reg.SharedPool,
		"APPMV3515": &appmv3reg.SharedPool,
		"APPMV3505": &appmv3reg.SharedPool,
		"IOSMESSREG": &iosmessreg.SharedPool,
	}
}

func newRegSxxxWorkerContext(platform, proxyStr, countryCode string) (regSxxxWorkerContext, error) {
	switch strings.ToLower(strings.TrimSpace(platform)) {
	case instagram.PlatformS415:
		return s415reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS425:
		return s425reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS435:
		return s435reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS445:
		return s445reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS416:
		return s416reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS417:
		return s417reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS418:
		return s418reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS419:
		return s419reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS420:
		return s420reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS421:
		return s421reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS422:
		return s422reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS423:
		return s423reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS424:
		return s424reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS426:
		return s426reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS427:
		return s427reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS428:
		return s428reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS429:
		return s429reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS430:
		return s430reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS431:
		return s431reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS432:
		return s432reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS433:
		return s433reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS434:
		return s434reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS436:
		return s436reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS437:
		return s437reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS438:
		return s438reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS439:
		return s439reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS440:
		return s440reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS441:
		return s441reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS442:
		return s442reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS443:
		return s443reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS444:
		return s444reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS446:
		return s446reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS447:
		return s447reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS448:
		return s448reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS449:
		return s449reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS450:
		return s450reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS451:
		return s451reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS452:
		return s452reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS453:
		return s453reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS454:
		return s454reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS455:
		return s455reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS456:
		return s456reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS457:
		return s457reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS458:
		return s458reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS459:
		return s459reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS460:
		return s460reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS461:
		return s461reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS462:
		return s462reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS463:
		return s463reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS464:
		return s464reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS465:
		return s465reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS466:
		return s466reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS467:
		return s467reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS468:
		return s468reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS469:
		return s469reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS470:
		return s470reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS471:
		return s471reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS472:
		return s472reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS473:
		return s473reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS474:
		return s474reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS475:
		return s475reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS476:
		return s476reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS477:
		return s477reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS478:
		return s478reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS479:
		return s479reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS480:
		return s480reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS481:
		return s481reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS482:
		return s482reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS483:
		return s483reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS484:
		return s484reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS485:
		return s485reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS486:
		return s486reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS487:
		return s487reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS488:
		return s488reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS489:
		return s489reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS490:
		return s490reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS491:
		return s491reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS492:
		return s492reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS493:
		return s493reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS494:
		return s494reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS496:
		return s496reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS497:
		return s497reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS498:
		return s498reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS499:
		return s499reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS495:
		return s495reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS500:
		return s500reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS501:
		return s501reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS502:
		return s502reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS503:
		return s503reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS504:
		return s504reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS505:
		return s505reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS506:
		return s506reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS507:
		return s507reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS508:
		return s508reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS509:
		return s509reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS510:
		return s510reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS511:
		return s511reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS512:
		return s512reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS513:
		return s513reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS514:
		return s514reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS515:
		return s515reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS516:
		return s516reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS517:
		return s517reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS518:
		return s518reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS519:
		return s519reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS520:
		return s520reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS521:
		return s521reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS522:
		return s522reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS523:
		return s523reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS524:
		return s524reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS525:
		return s525reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS526:
		return s526reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS527:
		return s527reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS528:
		return s528reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS529:
		return s529reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS530:
		return s530reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS531:
		return s531reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS532:
		return s532reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS533:
		return s533reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS534:
		return s534reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS535:
		return s535reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS536:
		return s536reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS537:
		return s537reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS538:
		return s538reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS539:
		return s539reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS540:
		return s540reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS541:
		return s541reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS542:
		return s542reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS543:
		return s543reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS544:
		return s544reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS545:
		return s545reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS546:
		return s546reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS547:
		return s547reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS548:
		return s548reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS549:
		return s549reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS550:
		return s550reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS551:
		return s551reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS552:
		return s552reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS553:
		return s553reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS554:
		return s554reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS555:
		return s555reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS555V2:
		return s555v2reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS556:
		return s556reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS557:
		return s557reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS558:
		return s558reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS558V2:
		return s558v2reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS559:
		return s559reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS559V2:
		return s559v2reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS560:
		return s560reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS560V2:
		return s560v2reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS561:
		return s561reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS561V2:
		return s561v2reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS561V3:
		return s561v3reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS561V99:
		return s561v99reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS561V4S21:
		return s561v4s21reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS561V4S23:
		return s561v4s23reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS562:
		return s562reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS562V3:
		return s562v3reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS562V4S21:
		return s562v4s21reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS562V4S23:
		return s562v4s23reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS563:
		return s563reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS563S21:
		return s563s21reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS563V3S21:
		return s563v3s21reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS563V4S21:
		return s563v4s21reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS563V4S23:
		return s563v4s23reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS563V5S21:
		return s563v5s21reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS563V5S23:
		return s563v5s23reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS563V6S21:
		return s563v6s21reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS563V6S23:
		return s563v6s23reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS564V1S21:
		return s564v1s21reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS564V1S23:
		return s564v1s23reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS564V2S21:
		return s564v2s21reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS564V2S23:
		return s564v2s23reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS564V3S21:
		return s564v3s21reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS564V3S23:
		return s564v3s23reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS565S21:
		return s565s21reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS565S23:
		return s565s23reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS565V2S21:
		return s565v2s21reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformS565V2S23:
		return s565v2s23reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformAppMV3:
		return appmv3reg.NewWorkerContext(proxyStr, countryCode)
	case instagram.PlatformAppMV3_535:
		return appmv3reg.NewWorkerContextPlatform(proxyStr, countryCode, "appmv3535")
	case instagram.PlatformAppMV3_545:
		return appmv3reg.NewWorkerContextPlatform(proxyStr, countryCode, "appmv3545")
	case instagram.PlatformAppMV3_555:
		return appmv3reg.NewWorkerContextPlatform(proxyStr, countryCode, "appmv3555")
	case instagram.PlatformAppMV3_563:
		return appmv3reg.NewWorkerContextPlatform(proxyStr, countryCode, "appmv3563")
	case instagram.PlatformAppMV3_564:
		return appmv3reg.NewWorkerContextPlatform(proxyStr, countryCode, "appmv3564")
	case instagram.PlatformAppMV3_565:
		return appmv3reg.NewWorkerContextPlatform(proxyStr, countryCode, "appmv3565")
	case instagram.PlatformAppMV3_525:
		return appmv3reg.NewWorkerContextPlatform(proxyStr, countryCode, "appmv3525")
	case instagram.PlatformAppMV3_515:
		return appmv3reg.NewWorkerContextPlatform(proxyStr, countryCode, "appmv3515")
	case instagram.PlatformAppMV3_505:
		return appmv3reg.NewWorkerContextPlatform(proxyStr, countryCode, "appmv3505")
	case instagram.PlatformAppMV3_490:
		return appmv3reg.NewWorkerContextPlatform(proxyStr, countryCode, "appmv3490")
	case instagram.PlatformIOSMessReg:
		return iosmessreg.NewWorkerContext(proxyStr, countryCode)
	default:
		return nil, fmt.Errorf("unsupported register platform %q", platform)
	}
}

func originalUABaseForSxxx(platform string) string {
	switch strings.ToLower(strings.TrimSpace(platform)) {
	case instagram.PlatformS415:
		return s415reg.OriginalUA
	case instagram.PlatformS425:
		return s425reg.OriginalUA
	case instagram.PlatformS435:
		return s435reg.OriginalUA
	case instagram.PlatformS445:
		return s445reg.OriginalUA
	case instagram.PlatformS416:
		return s416reg.OriginalUA
	case instagram.PlatformS417:
		return s417reg.OriginalUA
	case instagram.PlatformS418:
		return s418reg.OriginalUA
	case instagram.PlatformS419:
		return s419reg.OriginalUA
	case instagram.PlatformS420:
		return s420reg.OriginalUA
	case instagram.PlatformS421:
		return s421reg.OriginalUA
	case instagram.PlatformS422:
		return s422reg.OriginalUA
	case instagram.PlatformS423:
		return s423reg.OriginalUA
	case instagram.PlatformS424:
		return s424reg.OriginalUA
	case instagram.PlatformS426:
		return s426reg.OriginalUA
	case instagram.PlatformS427:
		return s427reg.OriginalUA
	case instagram.PlatformS428:
		return s428reg.OriginalUA
	case instagram.PlatformS429:
		return s429reg.OriginalUA
	case instagram.PlatformS430:
		return s430reg.OriginalUA
	case instagram.PlatformS431:
		return s431reg.OriginalUA
	case instagram.PlatformS432:
		return s432reg.OriginalUA
	case instagram.PlatformS433:
		return s433reg.OriginalUA
	case instagram.PlatformS434:
		return s434reg.OriginalUA
	case instagram.PlatformS436:
		return s436reg.OriginalUA
	case instagram.PlatformS437:
		return s437reg.OriginalUA
	case instagram.PlatformS438:
		return s438reg.OriginalUA
	case instagram.PlatformS439:
		return s439reg.OriginalUA
	case instagram.PlatformS440:
		return s440reg.OriginalUA
	case instagram.PlatformS441:
		return s441reg.OriginalUA
	case instagram.PlatformS442:
		return s442reg.OriginalUA
	case instagram.PlatformS443:
		return s443reg.OriginalUA
	case instagram.PlatformS444:
		return s444reg.OriginalUA
	case instagram.PlatformS446:
		return s446reg.OriginalUA
	case instagram.PlatformS447:
		return s447reg.OriginalUA
	case instagram.PlatformS448:
		return s448reg.OriginalUA
	case instagram.PlatformS449:
		return s449reg.OriginalUA
	case instagram.PlatformS450:
		return s450reg.OriginalUA
	case instagram.PlatformS451:
		return s451reg.OriginalUA
	case instagram.PlatformS452:
		return s452reg.OriginalUA
	case instagram.PlatformS453:
		return s453reg.OriginalUA
	case instagram.PlatformS454:
		return s454reg.OriginalUA
	case instagram.PlatformS455:
		return s455reg.OriginalUA
	case instagram.PlatformS456:
		return s456reg.OriginalUA
	case instagram.PlatformS457:
		return s457reg.OriginalUA
	case instagram.PlatformS458:
		return s458reg.OriginalUA
	case instagram.PlatformS459:
		return s459reg.OriginalUA
	case instagram.PlatformS460:
		return s460reg.OriginalUA
	case instagram.PlatformS461:
		return s461reg.OriginalUA
	case instagram.PlatformS462:
		return s462reg.OriginalUA
	case instagram.PlatformS463:
		return s463reg.OriginalUA
	case instagram.PlatformS464:
		return s464reg.OriginalUA
	case instagram.PlatformS465:
		return s465reg.OriginalUA
	case instagram.PlatformS466:
		return s466reg.OriginalUA
	case instagram.PlatformS467:
		return s467reg.OriginalUA
	case instagram.PlatformS468:
		return s468reg.OriginalUA
	case instagram.PlatformS469:
		return s469reg.OriginalUA
	case instagram.PlatformS470:
		return s470reg.OriginalUA
	case instagram.PlatformS471:
		return s471reg.OriginalUA
	case instagram.PlatformS472:
		return s472reg.OriginalUA
	case instagram.PlatformS473:
		return s473reg.OriginalUA
	case instagram.PlatformS474:
		return s474reg.OriginalUA
	case instagram.PlatformS475:
		return s475reg.OriginalUA
	case instagram.PlatformS476:
		return s476reg.OriginalUA
	case instagram.PlatformS477:
		return s477reg.OriginalUA
	case instagram.PlatformS478:
		return s478reg.OriginalUA
	case instagram.PlatformS479:
		return s479reg.OriginalUA
	case instagram.PlatformS480:
		return s480reg.OriginalUA
	case instagram.PlatformS481:
		return s481reg.OriginalUA
	case instagram.PlatformS482:
		return s482reg.OriginalUA
	case instagram.PlatformS483:
		return s483reg.OriginalUA
	case instagram.PlatformS484:
		return s484reg.OriginalUA
	case instagram.PlatformS485:
		return s485reg.OriginalUA
	case instagram.PlatformS486:
		return s486reg.OriginalUA
	case instagram.PlatformS487:
		return s487reg.OriginalUA
	case instagram.PlatformS488:
		return s488reg.OriginalUA
	case instagram.PlatformS489:
		return s489reg.OriginalUA
	case instagram.PlatformS490:
		return s490reg.OriginalUA
	case instagram.PlatformS491:
		return s491reg.OriginalUA
	case instagram.PlatformS492:
		return s492reg.OriginalUA
	case instagram.PlatformS493:
		return s493reg.OriginalUA
	case instagram.PlatformS494:
		return s494reg.OriginalUA
	case instagram.PlatformS496:
		return s496reg.OriginalUA
	case instagram.PlatformS497:
		return s497reg.OriginalUA
	case instagram.PlatformS498:
		return s498reg.OriginalUA
	case instagram.PlatformS499:
		return s499reg.OriginalUA
	case instagram.PlatformS495:
		return s495reg.OriginalUA
	case instagram.PlatformS500:
		return s500reg.OriginalUA
	case instagram.PlatformS501:
		return s501reg.OriginalUA
	case instagram.PlatformS502:
		return s502reg.OriginalUA
	case instagram.PlatformS503:
		return s503reg.OriginalUA
	case instagram.PlatformS504:
		return s504reg.OriginalUA
	case instagram.PlatformS505:
		return s505reg.OriginalUA
	case instagram.PlatformS506:
		return s506reg.OriginalUA
	case instagram.PlatformS507:
		return s507reg.OriginalUA
	case instagram.PlatformS508:
		return s508reg.OriginalUA
	case instagram.PlatformS509:
		return s509reg.OriginalUA
	case instagram.PlatformS510:
		return s510reg.OriginalUA
	case instagram.PlatformS511:
		return s511reg.OriginalUA
	case instagram.PlatformS512:
		return s512reg.OriginalUA
	case instagram.PlatformS513:
		return s513reg.OriginalUA
	case instagram.PlatformS514:
		return s514reg.OriginalUA
	case instagram.PlatformS515:
		return s515reg.OriginalUA
	case instagram.PlatformS516:
		return s516reg.OriginalUA
	case instagram.PlatformS517:
		return s517reg.OriginalUA
	case instagram.PlatformS518:
		return s518reg.OriginalUA
	case instagram.PlatformS519:
		return s519reg.OriginalUA
	case instagram.PlatformS520:
		return s520reg.OriginalUA
	case instagram.PlatformS521:
		return s521reg.OriginalUA
	case instagram.PlatformS522:
		return s522reg.OriginalUA
	case instagram.PlatformS523:
		return s523reg.OriginalUA
	case instagram.PlatformS524:
		return s524reg.OriginalUA
	case instagram.PlatformS525:
		return s525reg.OriginalUA
	case instagram.PlatformS526:
		return s526reg.OriginalUA
	case instagram.PlatformS527:
		return s527reg.OriginalUA
	case instagram.PlatformS528:
		return s528reg.OriginalUA
	case instagram.PlatformS529:
		return s529reg.OriginalUA
	case instagram.PlatformS530:
		return s530reg.OriginalUA
	case instagram.PlatformS531:
		return s531reg.OriginalUA
	case instagram.PlatformS532:
		return s532reg.OriginalUA
	case instagram.PlatformS533:
		return s533reg.OriginalUA
	case instagram.PlatformS534:
		return s534reg.OriginalUA
	case instagram.PlatformS535:
		return s535reg.OriginalUA
	case instagram.PlatformS536:
		return s536reg.OriginalUA
	case instagram.PlatformS537:
		return s537reg.OriginalUA
	case instagram.PlatformS538:
		return s538reg.OriginalUA
	case instagram.PlatformS539:
		return s539reg.OriginalUA
	case instagram.PlatformS540:
		return s540reg.OriginalUA
	case instagram.PlatformS541:
		return s541reg.OriginalUA
	case instagram.PlatformS542:
		return s542reg.OriginalUA
	case instagram.PlatformS543:
		return s543reg.OriginalUA
	case instagram.PlatformS544:
		return s544reg.OriginalUA
	case instagram.PlatformS545:
		return s545reg.OriginalUA
	case instagram.PlatformS546:
		return s546reg.OriginalUA
	case instagram.PlatformS547:
		return s547reg.OriginalUA
	case instagram.PlatformS548:
		return s548reg.OriginalUA
	case instagram.PlatformS549:
		return s549reg.OriginalUA
	case instagram.PlatformS550:
		return s550reg.OriginalUA
	case instagram.PlatformS551:
		return s551reg.OriginalUA
	case instagram.PlatformS552:
		return s552reg.OriginalUA
	case instagram.PlatformS553:
		return s553reg.OriginalUA
	case instagram.PlatformS554:
		return s554reg.OriginalUA
	case instagram.PlatformS555:
		return s555reg.OriginalUA
	case instagram.PlatformS555V2:
		return s555v2reg.OriginalUA
	case instagram.PlatformS556:
		return s556reg.OriginalUA
	case instagram.PlatformS557:
		return s557reg.OriginalUA
	case instagram.PlatformS558:
		return s558reg.OriginalUA
	case instagram.PlatformS558V2:
		return s558v2reg.OriginalUA
	case instagram.PlatformS559:
		return s559reg.OriginalUA
	case instagram.PlatformS559V2:
		return s559v2reg.OriginalUA
	case instagram.PlatformS560:
		return s560reg.OriginalUA
	case instagram.PlatformS560V2:
		return s560v2reg.OriginalUA
	case instagram.PlatformS561:
		return s561reg.OriginalUA
	case instagram.PlatformS561V2:
		return s561v2reg.OriginalUA
	case instagram.PlatformS561V3:
		return s561v3reg.OriginalUA
	case instagram.PlatformS561V99:
		return s561v99reg.OriginalUA
	case instagram.PlatformS561V4S21:
		return s561v4s21reg.OriginalUA
	case instagram.PlatformS561V4S23:
		return s561v4s23reg.OriginalUA
	case instagram.PlatformS562:
		return s562reg.OriginalUA
	case instagram.PlatformS562V3:
		return s562v3reg.OriginalUA
	case instagram.PlatformS562V4S21:
		return s562v4s21reg.OriginalUA
	case instagram.PlatformS562V4S23:
		return s562v4s23reg.OriginalUA
	case instagram.PlatformS563:
		return s563reg.OriginalUA
	case instagram.PlatformS563S21:
		return s563s21reg.OriginalUA
	case instagram.PlatformS563V3S21:
		return s563v3s21reg.OriginalUA
	case instagram.PlatformS563V4S21:
		return s563v4s21reg.OriginalUA
	case instagram.PlatformS563V4S23:
		return s563v4s23reg.OriginalUA
	case instagram.PlatformS563V5S21:
		return s563v5s21reg.OriginalUA
	case instagram.PlatformS563V5S23:
		return s563v5s23reg.OriginalUA
	case instagram.PlatformS563V6S21:
		return s563v6s21reg.OriginalUA
	case instagram.PlatformS563V6S23:
		return s563v6s23reg.OriginalUA
	case instagram.PlatformS564V1S21:
		return s564v1s21reg.OriginalUA
	case instagram.PlatformS564V1S23:
		return s564v1s23reg.OriginalUA
	case instagram.PlatformS564V2S21:
		return s564v2s21reg.OriginalUA
	case instagram.PlatformS564V2S23:
		return s564v2s23reg.OriginalUA
	case instagram.PlatformS564V3S21:
		return s564v3s21reg.OriginalUA
	case instagram.PlatformS564V3S23:
		return s564v3s23reg.OriginalUA
	case instagram.PlatformS565S21:
		return s565s21reg.OriginalUA
	case instagram.PlatformS565S23:
		return s565s23reg.OriginalUA
	case instagram.PlatformS565V2S21:
		return s565v2s21reg.OriginalUA
	case instagram.PlatformS565V2S23:
		return s565v2s23reg.OriginalUA
	case instagram.PlatformAppMV3, instagram.PlatformAppMV3_535, instagram.PlatformAppMV3_545,
		instagram.PlatformAppMV3_555, instagram.PlatformAppMV3_563, instagram.PlatformAppMV3_564,
		instagram.PlatformAppMV3_565, instagram.PlatformAppMV3_525, instagram.PlatformAppMV3_515, instagram.PlatformAppMV3_505, instagram.PlatformAppMV3_490:
		return appmv3reg.OriginalUA
	default:
		return ""
	}
}
