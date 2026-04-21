package message

import (
	"fmt"
	"strings"

	"copernico/deploy-notify-bot/internal/tag"
)

const descripcionPrefix = "· "

// Payload es el cuerpo JSON que se postea al webhook. Formato simple `{"text":"..."}`
// — compatible con relays tipo webhookbot.c-toss.com y con markdown en Teams.
type Payload struct {
	Text string `json:"text"`
}

type Input struct {
	Service     string
	Branch      string
	Release     string
	Description string
	Requested   []string
	Succeeded   []string
	PipelinesOK bool
}

// Build arma el payload para el webhook eligiendo el escenario según el estado.
func Build(in Input) Payload {
	if !in.PipelinesOK {
		return scenario5(in)
	}
	failed := diff(in.Requested, in.Succeeded)
	switch {
	case len(failed) == 0:
		return scenario1(in)
	case len(in.Succeeded) == 0:
		return scenario4(in, failed)
	default:
		return scenarioPartial(in, failed)
	}
}

func scenario1(in Input) Payload {
	return render(in, upperCSV(in.Succeeded), "OK",
		"Todos los ambientes fueron actualizados correctamente.")
}

func scenarioPartial(in Input, failed []string) Payload {
	return render(in, upperCSV(in.Succeeded), "OK",
		upperCSV(failed)+" no pudo ser actualizado.")
}

func scenario4(in Input, failed []string) Payload {
	return render(in, "", "OK",
		upperCSV(failed)+" no pudo ser actualizado.")
}

func scenario5(in Input) Payload {
	return render(in, "", "ERROR",
		"Error al ejecutar los pipelines.")
}

func render(in Input, ambientes, pipelines, aviso string) Payload {
	body := fmt.Sprintf(
		"SERVICIO: %s\nRAMA: %s\nDESCRIPCIÓN: %s%s\nRELEASE: %s\nAMBIENTES: %s\nPIPELINES: %s\nAVISO: %s",
		in.Service,
		strings.ToUpper(in.Branch),
		descripcionPrefix, in.Description,
		in.Release,
		ambientes,
		pipelines,
		aviso,
	)
	return Payload{Text: "```\n" + body + "\n```"}
}

func diff(requested, succeeded []string) []string {
	ok := map[string]bool{}
	for _, s := range succeeded {
		ok[s] = true
	}
	var failed []string
	for _, e := range requested {
		if !ok[e] {
			failed = append(failed, e)
		}
	}
	return tag.SortCanonical(failed)
}

func upperCSV(envs []string) string {
	sorted := tag.SortCanonical(envs)
	u := make([]string, 0, len(sorted))
	for _, e := range sorted {
		u = append(u, strings.ToUpper(e))
	}
	return strings.Join(u, ", ")
}
