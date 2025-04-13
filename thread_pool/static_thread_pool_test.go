package thread_pool

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type staticThreadPoolTestSuite struct {
	suite.Suite
	assert *assert.Assertions
}

func (suite *staticThreadPoolTestSuite) SetupTest() {
}

func (suite *staticThreadPoolTestSuite) TestCreate() {
	suite.assert = assert.New(suite.T())

	tp := NewStaticThreadPool(0)
	suite.assert.Nil(tp)

	tp = NewStaticThreadPool(1)
	suite.assert.NotNil(tp)

	tp = NewStaticThreadPool(1)
	suite.assert.NotNil(tp)
	suite.assert.Equal(tp.worker, uint32(1))
}

func (suite *staticThreadPoolTestSuite) TestStartStop() {
	suite.assert = assert.New(suite.T())

	tp := NewStaticThreadPool(2)
	suite.assert.NotNil(tp)
	suite.assert.Equal(tp.worker, uint32(2))

	tp.Start()
	suite.assert.NotNil(tp.priorityCh)
	suite.assert.NotNil(tp.normalCh)

	tp.Stop()
}

func (suite *staticThreadPoolTestSuite) TestSchedule() {
	suite.assert = assert.New(suite.T())

	tp := NewStaticThreadPool(2)
	suite.assert.NotNil(tp)
	suite.assert.Equal(tp.worker, uint32(2))

	tp.Start()
	suite.assert.NotNil(tp.priorityCh)
	suite.assert.NotNil(tp.normalCh)

	tp.Schedule(false, &PrefetchTask{failCnt: 1})
	tp.Schedule(true, &PrefetchTask{failCnt: 1})

	time.Sleep(1 * time.Second)
	tp.Stop()
}

type testTask struct {
}

var callbackCnt int32

func (t *testTask) Execute() {
	atomic.AddInt32(&callbackCnt, 1)
}

func (suite *staticThreadPoolTestSuite) TestPrioritySchedule() {
	suite.assert = assert.New(suite.T())

	callbackCnt = int32(0)

	tp := NewStaticThreadPool(10)
	suite.assert.NotNil(tp)
	suite.assert.Equal(tp.worker, uint32(10))

	tp.Start()
	suite.assert.NotNil(tp.priorityCh)
	suite.assert.NotNil(tp.normalCh)

	for i := 0; i < 100; i++ {
		tp.Schedule(i < 20, &testTask{})
	}

	time.Sleep(1 * time.Second)
	suite.assert.Equal(callbackCnt, int32(100))
	tp.Stop()
}

func TestThreadPoolSuite(t *testing.T) {
	suite.Run(t, new(staticThreadPoolTestSuite))
}
