package meta

import (
	"context"
	"errors"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/aws/retry"
	efsTypes "github.com/aws/aws-sdk-go-v2/service/efs/types"

	elasticacheTypes "github.com/aws/aws-sdk-go-v2/service/elasticache/types"
	secretsmanagerTypes "github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
	"github.com/aws/smithy-go"
	smithyhttp "github.com/aws/smithy-go/transport/http"
	"github.com/kyma-project/cloud-manager/pkg/composed"
	"github.com/kyma-project/cloud-manager/pkg/util"
)

type awsAccountKeyType struct{}

var awsAccountKey = awsAccountKeyType{}

func GetAwsAccountId(ctx context.Context) string {
	x := ctx.Value(awsAccountKey)
	s, ok := x.(string)
	if ok {
		return s
	}
	return ""
}

func SetAwsAccountId(ctx context.Context, val string) context.Context {
	return context.WithValue(ctx, awsAccountKey, val)
}

var retryStandard = retry.NewStandard()

func IsErrorRetryable(err error) bool {
	if err == nil {
		return false
	}
	return retryStandard.IsErrorRetryable(err)
}

func AsApiError(err error) smithy.APIError {
	var apiError smithy.APIError
	if errors.As(err, &apiError) {
		return apiError
	}
	return nil
}

func GetErrorMessage(err error) string {
	var apiError smithy.APIError
	if errors.As(err, &apiError) {
		switch apiError.ErrorCode() {
		case "UnauthorizedOperation":
			return "You are not authorized to perform this operation."
		}
		return apiError.ErrorMessage()
	}
	return err.Error()
}

func IsUnauthorized(err error) bool {
	var apiError smithy.APIError
	if errors.As(err, &apiError) {
		return apiError.ErrorCode() == "UnauthorizedOperation"
	}
	return false
}

var notFoundErrorCodes = map[string]struct{}{
	(&efsTypes.FileSystemNotFound{}).ErrorCode():                    {},
	(&efsTypes.AccessPointNotFound{}).ErrorCode():                   {},
	(&efsTypes.MountTargetNotFound{}).ErrorCode():                   {},
	(&efsTypes.PolicyNotFound{}).ErrorCode():                        {},
	(&elasticacheTypes.CacheSubnetGroupNotFoundFault{}).ErrorCode(): {},
	(&elasticacheTypes.CacheClusterNotFoundFault{}).ErrorCode():     {},
	(&secretsmanagerTypes.ResourceNotFoundException{}).ErrorCode():  {},
	"InvalidVpcPeeringConnectionID.NotFound":                        {},
}

func IsNotFound(err error) bool {
	if err != nil {
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) {
			var smithyhttpErr *smithyhttp.ResponseError

			_, listed := notFoundErrorCodes[apiErr.ErrorCode()]
			if listed {
				return true
			}

			if errors.As(err, &smithyhttpErr) {
				return smithyhttpErr.HTTPStatusCode() == http.StatusNotFound
			}

		}
	}
	return false
}

func RetryableErrorToRequeueResponse(err error) error {
	if IsErrorRetryable(err) {
		return composed.StopWithRequeueDelay(util.Timing.T10000ms())
	}
	return nil
}

func ErrorToRequeueResponse(err error) error {
	if err == nil {
		return nil
	}
	if IsErrorRetryable(err) {
		return composed.StopWithRequeueDelay(util.Timing.T10000ms())
	}
	return composed.StopWithRequeueDelay(util.Timing.T300000ms())
}

func LogErrorAndReturn(err error, msg string, ctx context.Context) (error, context.Context) {
	result := ErrorToRequeueResponse(err)
	return composed.LogErrorAndReturn(err, msg, result, ctx)
}

type ElastiCacheState = string

// github.com/aws/aws-sdk-go-v2/service/elasticache@v1.40.3/types/types.go
// Status *string
// The current state of this replication group - creating , available , modifying ,
// deleting , create-failed , snapshotting .
const (
	ElastiCache_AVAILABLE     ElastiCacheState = "available"
	ElastiCache_CREATING      ElastiCacheState = "creating"
	ElastiCache_DELETING      ElastiCacheState = "deleting"
	ElastiCache_MODIFYING     ElastiCacheState = "modifying"
	ElastiCache_CREATE_FAILED ElastiCacheState = "create-failed"
	ElastiCache_SNAPSHOTTING  ElastiCacheState = "snapshotting"
)

type ElastiCacheUserGroupState = string

const (
	ElastiCache_UserGroup_ACTIVE    ElastiCacheUserGroupState = "active"
	ElastiCache_UserGroup_CREATING  ElastiCacheUserGroupState = "creating"
	ElastiCache_UserGroup_DELETING  ElastiCacheUserGroupState = "deleting"
	ElastiCache_UserGroup_MODIFYING ElastiCacheUserGroupState = "modifying"
)
