package nsregistry

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/common/dynamicconfig"
	"go.temporal.io/server/common/log"
	"go.temporal.io/server/common/metrics"
	"go.temporal.io/server/common/namespace"
	"go.temporal.io/server/common/persistence"
	"go.uber.org/mock/gomock"
)

func TestRefreshNamespacesTimesOut(t *testing.T) {
	controller := gomock.NewController(t)
	regPersistence := NewMockPersistence(controller)
	regPersistence.EXPECT().ListNamespaces(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, _ *persistence.ListNamespacesRequest) (*persistence.ListNamespacesResponse, error) {
			<-ctx.Done()
			return nil, ctx.Err()
		},
	).AnyTimes()

	r := NewRegistry(
		regPersistence,
		true,
		"active",
		dynamicconfig.GetDurationPropertyFn(time.Second),
		dynamicconfig.GetBoolPropertyFn(false),
		metrics.NoopMetricsHandler,
		log.NewTestLogger(),
		namespace.NewDefaultReplicationResolverFactory(),
		DefaultNamespaceStateChanged,
	)

	start := time.Now()
	err := r.refreshNamespaces(context.Background())
	require.ErrorIs(t, err, context.DeadlineExceeded)
	// The refresh timeout is 10s; allow twice that for scheduling slack.
	require.Less(t, time.Since(start), 20*time.Second)
}
