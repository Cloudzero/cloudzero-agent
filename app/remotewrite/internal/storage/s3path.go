package storage

import (
	"fmt"
	"time"

	"github.com/go-obvious/env"
)

func CompressedFileExt() string {
	return "snappy"
}

func CompressedFilesPrefix(organizationID string) string {
	return fmt.Sprintf("raw_org_compressed/%s/container_metrics", organizationID)
}

func BucketName(organizationID string) string {
	return fmt.Sprintf("cz-%s-container-analysis-%s", env.MustGet("NAMESPACE"), organizationID)
}

func BuildPath(prefix string, t time.Time, cloudAccountID, clusterName, extension string) string {
	return fmt.Sprintf("%s/year=%s/month=%s/day=%s/hour=%s/cloud_account_id=%s/cluster_name=%s/%d.%s",
		prefix,
		GetYear(t),
		GetMonth(t),
		GetDay(t),
		GetHour(t),
		cloudAccountID,
		clusterName,
		t.UnixMilli(),
		extension,
	)
}

// DateParts contains the separate components of a date and time as strings.
type DateParts struct {
	Year  string // YYYY
	Month string // MM
	Day   string // DD
	Hour  string // HH
}

// GetYear extracts the year as a string with four digits.
func GetYear(t time.Time) string {
	return t.Format("2006")
}

// GetMonth extracts the month as a string with two digits (leading zero if necessary).
func GetMonth(t time.Time) string {
	return t.Format("01")
}

// GetDay extracts the day as a string with two digits (leading zero if necessary).
func GetDay(t time.Time) string {
	return t.Format("02")
}

// GetHour extracts the hour as a string with two digits (leading zero if necessary) in 24-hour format.
func GetHour(t time.Time) string {
	return t.Format("15")
}
