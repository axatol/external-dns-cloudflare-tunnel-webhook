package cf

import "github.com/cloudflare/cloudflare-go"

func RecordMapByID(records []cloudflare.DNSRecord) map[string]cloudflare.DNSRecord {
	result := make(map[string]cloudflare.DNSRecord, len(records))
	for _, record := range records {
		result[record.ID] = record
	}

	return result
}

func RecordMapByName(records []cloudflare.DNSRecord) map[string][]cloudflare.DNSRecord {
	result := map[string][]cloudflare.DNSRecord{}
	for _, record := range records {
		if _, ok := result[record.Name]; !ok {
			result[record.Name] = []cloudflare.DNSRecord{}
		}

		result[record.ID] = append(result[record.ID], record)
	}

	return result
}
