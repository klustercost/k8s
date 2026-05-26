package jsonata_ext

import (
	"strconv"
	"strings"

	jsonata "github.com/blues/jsonata-go"
)

//https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/#resource-units-in-kubernetes

func memory_quantity(src string) (uint64, error) {
	var multiplier uint64 = 1
	if strings.HasSuffix(src, "Ki") {
		src = strings.TrimSuffix(src, "Ki")
		multiplier = 1024
	} else if strings.HasSuffix(src, "Mi") {
		src = strings.TrimSuffix(src, "Mi")
		multiplier = 1024 * 1024
	} else if strings.HasSuffix(src, "Gi") {
		src = strings.TrimSuffix(src, "Gi")
		multiplier = 1024 * 1024 * 1024
	} else if strings.HasSuffix(src, "Ti") {
		src = strings.TrimSuffix(src, "Ti")
		multiplier = 1024 * 1024 * 1024 * 1024
	} else if strings.HasSuffix(src, "Pi") {
		src = strings.TrimSuffix(src, "Pi")
		multiplier = 1024 * 1024 * 1024 * 1024 * 1024
	} else if strings.HasSuffix(src, "Ei") {
		src = strings.TrimSuffix(src, "Ei")
		multiplier = 1024 * 1024 * 1024 * 1024 * 1024 * 1024
	} else if strings.HasSuffix(src, "K") {
		src = strings.TrimSuffix(src, "K")
		multiplier = 1000
	} else if strings.HasSuffix(src, "M") {
		src = strings.TrimSuffix(src, "M")
		multiplier = 1000 * 1000
	} else if strings.HasSuffix(src, "G") {
		src = strings.TrimSuffix(src, "G")
		multiplier = 1000 * 1000 * 1000
	} else if strings.HasSuffix(src, "T") {
		src = strings.TrimSuffix(src, "T")
		multiplier = 1000 * 1000 * 1000 * 1000
	} else if strings.HasSuffix(src, "P") {
		src = strings.TrimSuffix(src, "P")
		multiplier = 1000 * 1000 * 1000 * 1000 * 1000
	} else if strings.HasSuffix(src, "E") {
		src = strings.TrimSuffix(src, "E")
		multiplier = 1000 * 1000 * 1000 * 1000 * 1000 * 1000
	}

	intValue, err := strconv.Atoi(src)
	if err != nil {
		return 0, err
	}
	return uint64(intValue) * multiplier, nil
}

func cpu_quantity(src string) (float64, error) {
	divider := 1.0
	if strings.HasSuffix(src, "m") {
		src = strings.TrimSuffix(src, "m")
		divider = 1000.0
	}
	floatValue, err := strconv.ParseFloat(src, 64)
	if err != nil {
		return 0, err
	}
	return floatValue / divider, nil
}

var exts = map[string]jsonata.Extension{
	"memory_quantity": {
		Func: memory_quantity,
	},
	"cpu_quantity": {
		Func: cpu_quantity,
	},
}

func init() {
	jsonata.RegisterExts(exts)
}
