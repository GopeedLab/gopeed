package evaluation

import "testing"

func TestFrozenExtensionReplay(t *testing.T) {
	for _, name := range []string{"bilibili", "youtube", "twitter"} {
		t.Run(name, func(t *testing.T) {
			r := setup(t, "moejs-frozen", name)
			v, e := r.Eval("resolveOnce()")
			if e != nil {
				t.Fatal(e)
			}
			t.Log(v)
		})
	}
}
