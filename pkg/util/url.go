package util

import "strings"

const FileSchema = "FILE"

func ParseSchema(url string) string {
	index := strings.Index(url, ":")
	if index == -1 {
		return FileSchema
	}
	return strings.ToUpper(url[:index])
}
