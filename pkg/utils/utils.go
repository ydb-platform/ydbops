package utils

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"unicode"

	"github.com/ydb-platform/ydb-go-genproto/protos/Ydb"
	"github.com/ydb-platform/ydb-go-genproto/protos/Ydb_Issue"
	"github.com/ydb-platform/ydb-go-genproto/protos/Ydb_Operations"
	"go.uber.org/zap"

	"github.com/ydb-platform/ydbops/internal/collections"
)

func LogOperation(logger *zap.SugaredLogger, op *Ydb_Operations.Operation) {
	sb := strings.Builder{}
	sb.WriteString(fmt.Sprintf("Operation status: %s", op.Status))

	if len(op.Issues) > 0 {
		sb.WriteString(
			fmt.Sprintf("\nIssues:\n%s",
				strings.Join(collections.Convert(op.Issues,
					func(issue *Ydb_Issue.IssueMessage) string {
						return fmt.Sprintf("  Severity: %d, code: %d, message: %s", issue.Severity, issue.IssueCode, issue.Message)
					},
				), "\n"),
			))
	}

	if op.Status != Ydb.StatusIds_SUCCESS {
		logger.Errorf("GRPC invocation unsuccessful:\n%s", sb.String())
	} else {
		logger.Debugf("Invocation result:\n%s", sb.String())
	}
}

func ParseSSHArgs(rawArgs string) []string {
	args := []string{}
	isInsideQuotes := false

	rawRunes := []rune(rawArgs)
	curArg := []rune{}
	for i := 0; i < len(rawRunes); i++ {
		if rawRunes[i] == '\\' && i+1 < len(rawRunes) && rawRunes[i+1] == '"' {
			isInsideQuotes = !isInsideQuotes
			i++
			curArg = append(curArg, '"')
			continue
		}

		if unicode.IsSpace(rawRunes[i]) && !isInsideQuotes {
			if len(curArg) > 0 {
				args = append(args, string(curArg))
			}
			curArg = []rune{}
		} else {
			curArg = append(curArg, rawRunes[i])
		}
	}

	if len(curArg) > 0 {
		args = append(args, string(curArg))
	}

	return args
}

func GetNodeFQDNs(hostsRaw []string) ([]string, error) {
	hostFQDNs := make([]string, 0, len(hostsRaw))

	for _, hostFQDN := range hostsRaw {
		_, err := url.Parse(hostFQDN)
		if err != nil {
			return nil, fmt.Errorf("invalid host fqdn specified: %s", hostFQDN)
		}

		hostFQDNs = append(hostFQDNs, hostFQDN)
	}

	return hostsRaw, nil
}

func GetNodeIds(hosts []string) ([]uint32, error) {
	ids := make([]uint32, 0, len(hosts))

	for _, nodeID := range hosts {
		if strings.Contains(nodeID, "-") {
			rangeParts := strings.Split(nodeID, "-")
			start, err := strconv.Atoi(rangeParts[0])
			if err != nil {
				return nil, fmt.Errorf("failed to parse node id %v in range %s, %w", start, nodeID, err)
			}
			end, err := strconv.Atoi(rangeParts[1])
			if err != nil {
				return nil, fmt.Errorf("failed to parse node id %v in range %s, %w", end, nodeID, err)
			}
			for id := start; id <= end; id++ {
				ids = append(ids, uint32(id))
			}

			continue
		}

		id, err := strconv.Atoi(nodeID)
		if err != nil {
			return nil, fmt.Errorf("failed to parse node id: %w", err)
		}
		if id < 0 {
			return nil, fmt.Errorf("invalid node id specified: %d, must be positive", id)
		}
		ids = append(ids, uint32(id))
	}

	return ids, nil
}
