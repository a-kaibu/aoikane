package phrase

import (
	"github.com/a-kaibu/aoikane/internal/cabocha"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// Native contraction preserves sparse CaboCha ordinals and follows dependencies
// through removed chunks. These snapshots start from normal original text.
func TestDecimalChunkContractionOriginal(t *testing.T) {
	f := openFrontend(t)
	var err error
	f.Dependency, err = cabocha.LoadSVM(filepath.Join("..", "..", "assets", "cabocha", "standard-dep.ipa.model"))
	if err != nil {
		t.Fatal(err)
	}
	f.Bound = openBoundRules(t, "bound-rules-standard.txt")
	f.Predictor = openPredictor(t)
	for _, tc := range []struct {
		index int
		hash  string
	}{
		{0, "080236dd09a17f8c998381348cdb8c52b6488d35a0df348de65e7023f819655e"},
		{3, "3b9151e055f330ca0fc85d68ab1228d839d87c3298d8a6cae0702302f02f9e1b"},
		{5, "638532ed3581578fe6e3b2c273759712fd07a9108397ee8f55c4174ebd03a9a4"},
		{6, "6157ed131758b5db3f98ede9192a77233bec7dd61a33338b5b4687e58e110205"},
		{8, "9969f7ca9fcc14ed4544551e2837f7fbdebf61513dbcceef9daf339bef31b4bc"},
		{10, "a9d332a4f54aa3a8ceb58029822b04163f93a7de6dcec4fe9d79dd9e53ff1b18"},
		{11, "5bd3161f627b37667945ad12a86e43da81618779c1050bd072848758e302ab45"},
		{13, "477938811c59bac84fc297843c6e1cec65cb9449b82baebe7f0af922467f2ac7"},
		{24, "7f881a1d69096197ec0c67ef3cbe9cf2a21fb48a487c0e98eff010795b04a297"},
		{25, "24a5879792d1e111b3667883a8ae37fc987b47dd590049dda4e47a9215653d74"},
		{26, "e37b9b43203b0143c802ab3292802f8a1df7c124968210cce698df5167933fdf"},
		{27, "847bf3bea878976889f76a645399052cff10b9dd2456b53ebb8bd23263dc0d21"},
		{28, "94e2ee879dd933085ed849aec8419e58455f4f993e72216185adf0ab34b32a9e"},
		{29, "6cfbfb608a0a887b347931beb4bfb78c1f1bfde35d4bc434f8d7f53b23df8e72"},
		{30, "4b358a03117303d8b8479d1bfb6166649933fda09928bf873b7bc389abe51968"},
		{31, "6542e708cb5efb0047da76915bbb09523eaf3e34b5655c057cb52ed03fce7a74"},
		{1, "65a821866e9020d346de247990912cae47d5d8a4458dc0ade639ca715bf6e757"},
		{2, "f1c4072478e68e009cfe131937e2d62030e590b325e777440ca4b0e1ddf820b6"},
		{4, "81acba2b244cb29e635fecf1119d478062432fa4ef5eae388fa0d12ebb67db79"},
		{12, "fbd2b6f4e78acd9684d0e47ab32104e3461918385d578ece2290a5b3b54abac5"},
		{14, "6c7cbe442acfdad39af26ac422db1c8a5caa786f0fc26a323391a0878c2d5fa3"},
		{15, "8b178bff888518d98f61b2d4c9a9ccf105feb5c459f993ed94d2938c916a5c25"},
		{16, "b2c26edef063a151562cb60702e44613010386bd7b8695fa7bad6696983b8176"},
		{17, "2cdca5088d218b8a8eac89c71efddbc7cf238425ec53878fcd14c4ab37a1b888"},
		{18, "28deb8b0b05949c86b83a382a1fdd0cdd0962bb2c5825846e0ed238785a20579"},
		{19, "c820c6bb4b049a3db4bc8da0acec06f9f3583ccdadfcb9b57beb12ae01fd1251"},
		{20, "2d01cb1f70d7ec80a11b288d069af8d9332e4775c70939e46069c25d17a0052c"},
		{21, "951e2a45a9d63e5d2b8d91b0f252d419045b2884ec41cc8bca0a96eb672d4a7a"},
		{22, "e8ca25089129b399520d8dd9908da10f1164ebecc47eada9079c9c3227dc175c"},
		{23, "31e2f5c871d976e83416ae1f98a8341d6e005613bd31667d627fac2676f48927"},
		{32, "8ea27f5ce9b212be0d6182726e67d04cefc68bf49f500a570816ca4eb388576a"},
		{33, "2f0fe2a4a73fd2a7ed6c9549a47928b7bee2fb6f96052ccc03c836d28b98cb3a"},
		{34, "76bf8ec24954ec5221899d16efc7f33a9738593e95fd23aaa8220bc0a350636e"},
		{35, "67c0665cce9e944c48959145215443bd1dbeae9a14697008fd461ee6bb35b1de"},
		{36, "2a2e50745be1681e4ad4909f578c0f201fba5c24ff1485f9b085fb75d0bbb3e1"},
		{37, "59481ec560e874ac11158d3422a00f43455c8868ab85b5cfd38dc9f5a565f10e"},
		{38, "c565393c7c5602e7e439e7681848f62dfa7ed0b3f43a52583ce3db6150d18608"},
		{39, "c0f80c3aa0cd03a2679ed4d42701fcce6169bc59fe53154d90cf6d2598a6911f"},
		{40, "f1e32e11a138bfa2c84d31634a74cd30bfa6ad670f8abcc0a331b8663a467577"},
		{41, "60aedb148971c15b414e138ea63ce83366713e9e92aaaa077e46077121474bfa"},
		{42, "f3da1dfbc980766851f839f69f5f747b241e7ae54560d1f4da04e8cc5af1b86b"},
		{43, "91cba3a7893529136592ad0cd55d11bda163de72002ffe143dda576d45bb87ca"},
		{44, "208d7f4daaa8f2dc1b85b6afa2f3f128893580d97700ddd962652ddc9d258512"},
		{45, "89e979927df99201434350f6b7a18039c5f61e28c8678bd65f35f1a5f10955b5"},
		{46, "5121cb14f486846bb03e35dcab0d0dea7f5a46a87d014b9967d278558df0f612"},
		{47, "ef2897b1f290fe74de3de45394d18cc0c491260b8e15908ff6611930ab0bdf07"},
		{48, "7f368238d958e2404818e34bf7c6e2104203a08c0159ea609bd0346b3b369afb"},
		{49, "e031fee593ce6cbd89f5a09b808abb3652f75dc130ccb74ea1c966b3dde931a1"},
		{50, "8bdfc8a1baa0944dea0e5dbd11e72345d3b2c3edc0df94b51077af3e345e0108"},
		{51, "35af9eeeb9fccb7125d0d53939d6a78d47d5ad5e400c645de0f8445350df5038"},
		{52, "51a0e319c3de2f4ad12e4014a69043b233ee9027ae4c5db9a354a2486377b516"},
		{53, "c0b9bdbdf75b2ea7a8c44e0c666dc65c3eb574b97948824d389b1635d85aabd9"},
		{54, "9d6c4d873a0c4bbe50804fd87f56715265cedfd25a5008b276ba4806c840b963"},
		{55, "3401396c916369ae626095146ae45ae20c627d202ae86d606a3b8e8091453490"},
		{56, "02d5eda6df0fa85cd55cb7ba4952a1a69dfebbadf26b1ae5537a01f890059218"},
		{57, "39d278e5ffb3a9eda852a7cbf86f9e1e47c601d7ae750421936c2c3e1d95f132"},
	} {
		path := fmt.Sprintf("../../private/stages_decimal_chunks/c%02d.jsonl", tc.index)
		b, err := os.ReadFile(path)
		if os.IsNotExist(err) {
			t.Skip("private original decimal chunk snapshots unavailable")
		}
		if err != nil {
			t.Fatal(err)
		}
		if fmt.Sprintf("%x", sha256.Sum256(b)) != tc.hash {
			t.Fatal("original decimal chunk reference differs")
		}
		ss, err := ReadSnapshots(path)
		if err != nil {
			t.Fatal(err)
		}
		var native *Document
		text, wantKana := "", ""
		for _, s := range ss {
			if s.Event == "stage_leave_1b9f10" {
				native, err = s.Load(f.Codec)
				text = s.Input
				if err != nil {
					t.Fatal(err)
				}
			}
			if s.Event == "format" {
				wantKana, err = s.FormatText(f.Codec)
				if err != nil {
					t.Fatal(err)
				}
			}
		}
		if native == nil || wantKana == "" {
			t.Fatal("incomplete original decimal capture")
		}
		t.Run(text, func(t *testing.T) {
			got, err := f.Graph(text)
			if err != nil {
				t.Fatal(err)
			}
			if diff := compareGraphs(got, native); diff != "" {
				t.Fatal(diff)
			}
			if err := got.attachSingleSentence(); err != nil {
				t.Fatal(err)
			}
			if a, b := cutChunkRecords(t, got), cutChunkRecords(t, native); !reflect.DeepEqual(a, b) {
				t.Fatalf("Go chunks=%v original=%v", a, b)
			}
			kana, err := f.Kana(text)
			if err != nil {
				t.Fatal(err)
			}
			if kana != wantKana {
				t.Fatalf("kana=%s original=%s", kana, wantKana)
			}
		})
	}
}
