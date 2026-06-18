package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

var outWriter io.Writer = os.Stdout

func printUsageAndExit() {
	fmt.Fprintln(outWriter, "Usage:")
	fmt.Fprintln(outWriter, "  cistern clients create '<json_payload>'   (e.g., '{\"name\": \"my-client\"}')")
	fmt.Fprintln(outWriter, "  cistern clients read '<id_or_json>'       (e.g., 'some-uuid-here' or '{\"id\": \"some-uuid\"}')")
	fmt.Fprintln(outWriter, "  cistern clients update '<json_payload>'   (e.g., '{\"id\": \"uuid\", \"name\": \"new-name\"}')")
	fmt.Fprintln(outWriter, "  cistern clients delete '<id_or_json>'     (e.g., 'some-uuid-here' or '{\"id\": \"some-uuid\"}')")
	fmt.Fprintln(outWriter, "  cistern apikeys generate '<json_payload>' (e.g., '{\"client_id\": \"client-uuid\", \"name\": \"my-key\"}')")
	fmt.Fprintln(outWriter, "  cistern apikeys read '<id_or_json>'       (e.g., 'some-uuid-here' or '{\"id\": \"some-uuid\"}')")
	fmt.Fprintln(outWriter, "  cistern apikeys delete '<id_or_json>'     (e.g., 'some-uuid-here' or '{\"id\": \"some-uuid\"}')")
	fmt.Fprintln(outWriter, "  cistern buckets create '<json_payload>'   (e.g., '{\"bucket_key\": \"my-bucket\", \"owner_id\": \"client-uuid\"}')")
	fmt.Fprintln(outWriter, "  cistern buckets read '<id_or_json>'       (e.g., 'some-uuid-here' or '{\"id\": \"some-uuid\"}')")
	fmt.Fprintln(outWriter, "  cistern buckets edit '<json_payload>'     (e.g., '{\"id\": \"uuid\", \"bucket_key\": \"new-key\", \"owner_id\": \"client-uuid\"}')")
	fmt.Fprintln(outWriter, "  cistern buckets update '<json_payload>'   (e.g., '{\"id\": \"uuid\", \"bucket_key\": \"new-key\", \"owner_id\": \"client-uuid\"}')")
	fmt.Fprintln(outWriter, "  cistern buckets delete '<id_or_json>'     (e.g., 'some-uuid-here' or '{\"id\": \"some-uuid\"}')")
	fmt.Fprintln(outWriter, "  cistern objects upload '<json_payload>' <filepath> (e.g., '{\"bucket_id\": \"uuid\", \"object_key\": \"notes.txt\"}' /path/to/file.txt)")
	fmt.Fprintln(outWriter, "  cistern objects read '<id_or_json>'                (e.g., 'some-uuid-here')")
	fmt.Fprintln(outWriter, "  cistern objects download '<id_or_json>' <dest_path> (e.g., 'some-uuid-here' ./downloaded.txt)")
	fmt.Fprintln(outWriter, "  cistern objects delete '<id_or_json>'              (e.g., 'some-uuid-here')")
	fmt.Fprintln(outWriter, "  cistern objects list '<bucket_id_or_json>'         (e.g., 'bucket-uuid-here')")
	os.Exit(1)
}

func printJSON(v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON response: %w", err)
	}
	fmt.Fprintln(outWriter, string(data))
	return nil
}

func extractID(payload string) string {
	trimmed := strings.TrimSpace(payload)
	if strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}") {
		var data struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal([]byte(trimmed), &data); err == nil && data.ID != "" {
			return data.ID
		}
	}
	return trimmed
}
