// versions.go — Hằng số API version-specific cho VERIFY Messenger Android.
// Bảng riêng (không import từ register để tránh import cycle register↔verify).
// Đồng bộ với register/android/appmessv3/versions.go.
package appmessv3

type vver struct {
	platform    string
	docID       string // action client_doc_id (login + add-email + confirm + resend)
	renderDocID string // AppRootQuery render (bottomsheet/change_email)
	bloksVer    string // bloks_versioning_id
	fbav        string // FBAV cố định ("" = pool, chỉ 530)
	fbbv        string
}

var vvers = map[string]vver{
	"appmessv3": {
		platform:    "appmessv3",
		docID:       "11994080426288799937543572098",
		renderDocID: "10537346114757978319483558625",
		bloksVer:    "dadbbd68d34735f7a39b791542ad0ecd1b257eddf3e70ab790d47b3cedd8b093",
	},
	"appmessv3_535": {
		platform:    "appmessv3_535",
		docID:       "11994080425969031306509088180",
		renderDocID: "1053734612728726099723365008",
		bloksVer:    "8529dbd160b6773bacafe4db8da1b3e4f3f7f6a46325fdcc55ea121fcedbca99",
		fbav:        "535.0.0.101.107",
		fbbv:        "840054075",
	},
	"appmessv3_545": {
		platform:    "appmessv3_545",
		docID:       "119940804210985767242572828821",
		renderDocID: "1053734613076523724584234653",
		bloksVer:    "b985267426c80f5463f00406de5e8921f6663901e6baf002f252315aa5136d40",
		fbav:        "545.0.0.27.62",
		fbbv:        "870175947",
	},
	"appmessv3_555": {
		platform:    "appmessv3_555",
		docID:       "119940804213456669668634861156",
		renderDocID: "1053734614916294106308739152",
		bloksVer:    "9688c036938d7d39ef8ab0d37e3391ed20411bc4d34f3609a38cb763058168ca",
		fbav:        "555.0.0.56.66",
		fbbv:        "930834402",
	},
	"appmessv3_563": {
		platform:    "appmessv3_563",
		docID:       "11994080425265950746786919715",
		renderDocID: "10537346145421676708107647",
		bloksVer:    "0601ff3b92f35b99993dbf34a9e00545023fd12f7cd800ac976de998e777ebf1",
		fbav:        "563.0.0.47.86",
		fbbv:        "979328543",
	},
	"appmessv3_564": {
		platform:    "appmessv3_564",
		docID:       "11994080429256953651754343473",
		renderDocID: "10537346112485268848416167621",
		bloksVer:    "0601ff3b92f35b99993dbf34a9e00545023fd12f7cd800ac976de998e777ebf1",
		fbav:        "564.0.0.42.89",
		fbbv:        "984961990",
	},
	"appmessv3_565": {
		platform:    "appmessv3_565",
		docID:       "11994080421721054503625823822",
		renderDocID: "10537346110500195560705709374",
		bloksVer:    "0601ff3b92f35b99993dbf34a9e00545023fd12f7cd800ac976de998e777ebf1",
		fbav:        "565.0.0.0.2",
		fbbv:        "981799924",
	},
	"appmessv3_525": {
		platform:    "appmessv3_525",
		docID:       "11994080421627357484457011346",
		renderDocID: "10537346115290752020670458285",
		bloksVer:    "30a1b0d4f253b8b85aac1e6139d018b42404f8f5c4af913f7b8571f4765950f8",
		fbav:        "525.0.0.44.108",
		fbbv:        "792260954",
	},
	"appmessv3_515": {
		platform:    "appmessv3_515",
		docID:       "11994080429499736631360212705",
		renderDocID: "10537346112863072829216709694",
		bloksVer:    "f0ebfb9d4794d38a8e7075319ae6628295d75342284cd94abd2ccaad7402678b",
		fbav:        "515.0.0.51.108",
		fbbv:        "763707183",
	},
	"appmessv3_505": {
		platform:    "appmessv3_505",
		docID:       "119940804215590936760760473156",
		renderDocID: "1053734618828721923200278840",
		bloksVer:    "12af4c052783f63ec0360c80c69b3be74028764b443d7ca120f84775c4a6e124",
		fbav:        "505.0.0.62.82",
		fbbv:        "730961636",
	},
	"appmessv3_490": {
		platform:    "appmessv3_490",
		docID:       "11994080425481681952686497337",
		renderDocID: "1053734611015739896301870743",
		bloksVer:    "b1777b7e3a4aeefddcdc89a331dcfe8d3ae1a800e109c4ee92608f75c344318c",
		fbav:        "490.0.0.42.108",
		fbbv:        "684080902",
	},
}

func vverForPlatform(platform string) vver {
	if v, ok := vvers[platform]; ok {
		return v
	}
	return vvers["appmessv3"]
}
