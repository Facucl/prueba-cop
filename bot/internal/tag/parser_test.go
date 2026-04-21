package tag

import (
	"reflect"
	"testing"
)

func TestParseList(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want []string
	}{
		{"espacios", "desa test preprod", []string{"desa", "test", "preprod"}},
		{"coma", "desa,test,preprod", []string{"desa", "test", "preprod"}},
		{"mix case y separadores", "DESA, test  PREPROD", []string{"desa", "test", "preprod"}},
		{"subset", "preprod", []string{"preprod"}},
		{"prod se acepta", "prod", []string{"prod"}},
		{"todos incluido prod", "desa test preprod prod", []string{"desa", "test", "preprod", "prod"}},
		{"env desconocido va al final alfabético", "stage desa demo", []string{"desa", "demo", "stage"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseList(tc.in)
			if err != nil {
				t.Fatalf("err inesperado: %v", err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("want %v got %v", tc.want, got)
			}
		})
	}
	if _, err := ParseList(""); err == nil {
		t.Fatalf("string vacío debería dar error")
	}
}

func TestParseRequestedEnvs(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		want    []string
		wantErr bool
	}{
		{"todos en orden", "deploy: desa test preprod", []string{"desa", "test", "preprod"}, false},
		{"solo dos", "deploy: desa test", []string{"desa", "test"}, false},
		{"desordenado se normaliza", "deploy: preprod desa", []string{"desa", "preprod"}, false},
		{"mayúsculas", "DEPLOY: DESA TEST", []string{"desa", "test"}, false},
		{"separado por comas", "deploy: desa, test, preprod", []string{"desa", "test", "preprod"}, false},
		{"con otras líneas", "Mantenimiento: Fixes de seguridad y reparación automática de dependencias.\ndeploy: desa preprod\n", []string{"desa", "preprod"}, false},
		{"prod se acepta", "deploy: prod", []string{"prod"}, false},
		{"sin deploy", "solo texto", nil, true},
		{"deploy vacío", "deploy:", nil, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseRequestedEnvs(tc.in)
			if (err != nil) != tc.wantErr {
				t.Fatalf("wantErr=%v got err=%v", tc.wantErr, err)
			}
			if !tc.wantErr && !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("want %v got %v", tc.want, got)
			}
		})
	}
}

func TestSortCanonical(t *testing.T) {
	got := SortCanonical([]string{"preprod", "test", "desa", "prod"})
	want := []string{"desa", "test", "preprod", "prod"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v got %v", want, got)
	}

	got = SortCanonical([]string{"stage", "preprod", "alpha"})
	want = []string{"preprod", "alpha", "stage"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v got %v", want, got)
	}
}
