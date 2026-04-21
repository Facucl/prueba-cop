package message

import (
	"strings"
	"testing"
)

func fieldValue(text, name string) string {
	for _, line := range strings.Split(text, "\n") {
		prefix := name + ": "
		if strings.HasPrefix(line, prefix) {
			return strings.TrimPrefix(line, prefix)
		}
	}
	return ""
}

func TestBuildScenarios(t *testing.T) {
	base := Input{
		Service:     "ws-copernico-suite-back-cop",
		Branch:      "master",
		Release:     "release-1.0.33",
		Description: "Mantenimiento: Fixes de seguridad y reparación automática de dependencias.",
	}

	t.Run("1 exitoso", func(t *testing.T) {
		in := base
		in.PipelinesOK = true
		in.Requested = []string{"desa", "test", "preprod"}
		in.Succeeded = []string{"desa", "test", "preprod"}
		p := Build(in)
		if !strings.HasPrefix(p.Text, "```\n") || !strings.HasSuffix(p.Text, "\n```") {
			t.Fatalf("payload no está envuelto en code block: %q", p.Text)
		}
		if got := fieldValue(p.Text, "SERVICIO"); got != "ws-copernico-suite-back-cop" {
			t.Fatalf("SERVICIO inesperado: %q", got)
		}
		if got := fieldValue(p.Text, "RAMA"); got != "MASTER" {
			t.Fatalf("RAMA en mayúsculas esperada, got %q", got)
		}
		if got := fieldValue(p.Text, "DESCRIPCIÓN"); got != "· Mantenimiento: Fixes de seguridad y reparación automática de dependencias." {
			t.Fatalf("DESCRIPCIÓN inesperada: %q", got)
		}
		if got := fieldValue(p.Text, "AMBIENTES"); got != "DESA, TEST, PREPROD" {
			t.Fatalf("AMBIENTES inesperado: %q", got)
		}
		if got := fieldValue(p.Text, "PIPELINES"); got != "OK" {
			t.Fatalf("PIPELINES inesperado: %q", got)
		}
		if got := fieldValue(p.Text, "AVISO"); got != "Todos los ambientes fueron actualizados correctamente." {
			t.Fatalf("AVISO inesperado: %q", got)
		}
	})

	t.Run("2 falló TEST", func(t *testing.T) {
		in := base
		in.PipelinesOK = true
		in.Requested = []string{"desa", "test", "preprod"}
		in.Succeeded = []string{"desa", "preprod"}
		p := Build(in)
		if got := fieldValue(p.Text, "AMBIENTES"); got != "DESA, PREPROD" {
			t.Fatalf("AMBIENTES inesperado: %q", got)
		}
		if got := fieldValue(p.Text, "AVISO"); got != "TEST no pudo ser actualizado." {
			t.Fatalf("AVISO inesperado: %q", got)
		}
	})

	t.Run("3 fallaron DESA y TEST", func(t *testing.T) {
		in := base
		in.PipelinesOK = true
		in.Requested = []string{"desa", "test", "preprod"}
		in.Succeeded = []string{"preprod"}
		p := Build(in)
		if got := fieldValue(p.Text, "AMBIENTES"); got != "PREPROD" {
			t.Fatalf("AMBIENTES inesperado: %q", got)
		}
		if got := fieldValue(p.Text, "AVISO"); got != "DESA, TEST no pudo ser actualizado." {
			t.Fatalf("AVISO inesperado: %q", got)
		}
	})

	t.Run("4 fallaron todos los solicitados", func(t *testing.T) {
		in := base
		in.PipelinesOK = true
		in.Requested = []string{"desa", "test", "preprod"}
		in.Succeeded = []string{}
		p := Build(in)
		if got := fieldValue(p.Text, "AMBIENTES"); got != "" {
			t.Fatalf("AMBIENTES debería ser vacío: %q", got)
		}
		if got := fieldValue(p.Text, "PIPELINES"); got != "OK" {
			t.Fatalf("PIPELINES debería ser OK (pipeline anduvo, Argo no), got %q", got)
		}
		if got := fieldValue(p.Text, "AVISO"); got != "DESA, TEST, PREPROD no pudo ser actualizado." {
			t.Fatalf("AVISO inesperado: %q", got)
		}
	})

	t.Run("5 pipelines error", func(t *testing.T) {
		in := base
		in.PipelinesOK = false
		in.Requested = []string{"desa"}
		p := Build(in)
		if got := fieldValue(p.Text, "PIPELINES"); got != "ERROR" {
			t.Fatalf("PIPELINES debería ser ERROR, got %q", got)
		}
		if got := fieldValue(p.Text, "AMBIENTES"); got != "" {
			t.Fatalf("AMBIENTES debería ser vacío, got %q", got)
		}
		if got := fieldValue(p.Text, "AVISO"); got != "Error al ejecutar los pipelines." {
			t.Fatalf("AVISO inesperado: %q", got)
		}
	})

	t.Run("subset solicitado y todos OK es caso 1", func(t *testing.T) {
		in := base
		in.PipelinesOK = true
		in.Requested = []string{"desa", "test"}
		in.Succeeded = []string{"desa", "test"}
		p := Build(in)
		if got := fieldValue(p.Text, "AMBIENTES"); got != "DESA, TEST" {
			t.Fatalf("AMBIENTES inesperado: %q", got)
		}
		if got := fieldValue(p.Text, "AVISO"); got != "Todos los ambientes fueron actualizados correctamente." {
			t.Fatalf("AVISO inesperado: %q", got)
		}
	})

	t.Run("prod también se soporta", func(t *testing.T) {
		in := base
		in.PipelinesOK = true
		in.Requested = []string{"desa", "prod"}
		in.Succeeded = []string{"desa", "prod"}
		p := Build(in)
		if got := fieldValue(p.Text, "AMBIENTES"); got != "DESA, PROD" {
			t.Fatalf("AMBIENTES inesperado: %q", got)
		}
	})
}
