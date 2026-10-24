package xmlrpc

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestXMLRPCEncodesParametersAsValues(t *testing.T) {
	body, err := EncodeCall("d.directory.set", []any{"ABC", `/data/a, "b" & <c>`, 5, int64(1) << 40, true, []byte("x"), []any{"y"}})
	if err != nil {
		t.Fatal(err)
	}
	s := string(body)
	for _, want := range []string{
		`<methodName>d.directory.set</methodName>`,
		`<string>/data/a, &#34;b&#34; &amp; &lt;c&gt;</string>`,
		`<i4>5</i4>`,
		`<i8>1099511627776</i8>`,
		`<boolean>1</boolean>`,
		`<base64>eA==</base64>`,
		`<array><data><value><string>y</string></value></data></array>`,
	} {
		if !strings.Contains(s, want) {
			t.Errorf("call %s lacks %s", s, want)
		}
	}
}

func TestXMLRPCDecodesResponses(t *testing.T) {
	v, err := DecodeResponse([]byte(`<?xml version="1.0"?><methodResponse><params><param><value><array><data>` +
		`<value><i8>9007199254740993</i8></value><value><i4>-2</i4></value><value>plain</value><value><string>s</string></value>` +
		`<value><boolean>1</boolean></value><value><double>1.5</double></value><value><base64>eA==</base64></value>` +
		`<value><struct><member><name>k</name><value><int>3</int></value></member></struct></value>` +
		`</data></array></value></param></params></methodResponse>`))
	if err != nil {
		t.Fatal(err)
	}
	want := []any{int64(9007199254740993), int64(-2), "plain", "s", true, 1.5, []byte("x"), map[string]any{"k": int64(3)}}
	if !reflect.DeepEqual(v, want) {
		t.Fatalf("decoded %#v\nwant %#v", v, want)
	}
}

func TestXMLRPCFault(t *testing.T) {
	_, err := DecodeResponse([]byte(`<?xml version="1.0"?><methodResponse><fault><value><struct>` +
		`<member><name>faultCode</name><value><i8>-500</i8></value></member>` +
		`<member><name>faultString</name><value><string>invalid parameters: info-hash not found</string></value></member>` +
		`</struct></value></fault></methodResponse>`))
	var fault *Fault
	if !errors.As(err, &fault) || fault.Code != -500 || !strings.Contains(fault.Message, "not found") {
		t.Fatalf("fault %v", err)
	}
}

func TestXMLRPCAddressSchemes(t *testing.T) {
	for _, ok := range []string{"http://127.0.0.1/RPC2", "https://nas.example/RPC2", "scgi://127.0.0.1:5000", "scgi:///run/rtorrent/rpc.socket"} {
		if _, err := NewClient(ok, "", ""); err != nil {
			t.Errorf("%s: %v", ok, err)
		}
	}
	for _, bad := range []string{"", "ftp://host", "scgi://", "http:///RPC2"} {
		if _, err := NewClient(bad, "", ""); err == nil {
			t.Errorf("%q: want an error", bad)
		}
	}
}

func TestXMLRPCStructMembersInNameOrder(t *testing.T) {
	// rtorrent's system.multicall refuses a struct whose first member is not
	// methodName.
	for i := 0; i < 20; i++ {
		body, err := EncodeCall("system.multicall", []any{[]any{map[string]any{"params": []any{"H"}, "methodName": "d.name"}}})
		if err != nil {
			t.Fatal(err)
		}
		s := string(body)
		if strings.Index(s, "<name>methodName</name>") > strings.Index(s, "<name>params</name>") {
			t.Fatalf("methodName after params: %s", s)
		}
	}
}
