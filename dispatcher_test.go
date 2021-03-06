// Copyright (c) 2014 - Max Persson <max@looplab.se>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package eventhorizon

import (
	"fmt"
	"reflect"
	"time"

	. "gopkg.in/check.v1"

	t "github.com/looplab/eventhorizon/testing"
)

var _ = Suite(&DelegateDispatcherSuite{})
var _ = Suite(&ReflectDispatcherSuite{})
var _ = Suite(&DispatcherSuite{})

type DelegateDispatcherSuite struct {
	store *MockEventStore
	bus   *MockEventBus
	disp  *DelegateDispatcher
}

func (s *DelegateDispatcherSuite) SetUpTest(c *C) {
	s.store = &MockEventStore{
		events: make([]Event, 0),
	}
	s.bus = &MockEventBus{
		events: make([]Event, 0),
	}
	s.disp = NewDelegateDispatcher(s.store, s.bus)
}

func (s *DelegateDispatcherSuite) Test_NewDelegateAggregate(c *C) {
	store := &MockEventStore{
		events: make([]Event, 0),
	}
	bus := &MockEventBus{
		events: make([]Event, 0),
	}
	disp := NewDelegateDispatcher(store, bus)
	c.Assert(disp, Not(Equals), nil)
	c.Assert(disp.eventStore, Equals, store)
	c.Assert(disp.eventBus, Equals, bus)
	c.Assert(disp.commandHandlers, Not(Equals), nil)
}

var dispatchedDelegateCommand Command

type TestDelegateDispatcherAggregate struct {
	Aggregate
}

func (t *TestDelegateDispatcherAggregate) HandleCommand(command Command) ([]Event, error) {
	dispatchedDelegateCommand = command
	switch command := command.(type) {
	case TestCommand:
		if command.Content == "error" {
			return nil, fmt.Errorf("command error")
		}
		return []Event{TestEvent{command.TestID, command.Content}}, nil
	}
	return nil, fmt.Errorf("couldn't handle command")
}

func (t *TestDelegateDispatcherAggregate) HandleEvent(event Event) {
}

func (s *DelegateDispatcherSuite) Test_Dispatch_Simple(c *C) {
	aggregate := &TestDelegateDispatcherAggregate{}
	aggregateBaseType := reflect.ValueOf(aggregate).Elem().Type()
	s.disp.commandHandlers[reflect.TypeOf(TestCommand{})] = aggregateBaseType
	command1 := TestCommand{NewUUID(), "command1"}
	err := s.disp.Dispatch(command1)
	c.Assert(dispatchedDelegateCommand, Equals, command1)
	c.Assert(err, Equals, nil)
}

func (s *DelegateDispatcherSuite) Test_Dispatch_ErrorInHandler(c *C) {
	aggregate := &TestDelegateDispatcherAggregate{}
	aggregateBaseType := reflect.ValueOf(aggregate).Elem().Type()
	s.disp.commandHandlers[reflect.TypeOf(TestCommand{})] = aggregateBaseType
	commandError := TestCommand{NewUUID(), "error"}
	err := s.disp.Dispatch(commandError)
	c.Assert(err, ErrorMatches, "command error")
	c.Assert(dispatchedDelegateCommand, Equals, commandError)
}

func (s *DelegateDispatcherSuite) Test_Dispatch_NoHandlers(c *C) {
	command1 := TestCommand{NewUUID(), "command1"}
	err := s.disp.Dispatch(command1)
	c.Assert(err, ErrorMatches, "no handlers for command")
}

func (s *DelegateDispatcherSuite) Test_AddHandler_Simple(c *C) {
	aggregate := &TestDelegateDispatcherAggregate{}
	s.disp.AddHandler(aggregate, TestCommand{})
	c.Assert(len(s.disp.commandHandlers), Equals, 1)
	commandType := reflect.TypeOf(TestCommand{})
	c.Assert(s.disp.commandHandlers, t.HasKey, commandType)
	aggregateBaseType := reflect.ValueOf(aggregate).Elem().Type()
	c.Assert(s.disp.commandHandlers[commandType], Equals, aggregateBaseType)
}

func (s *DelegateDispatcherSuite) Test_AddHandler_Duplicate(c *C) {
	aggregate := &TestDelegateDispatcherAggregate{}
	s.disp.AddHandler(aggregate, TestCommand{})
	aggregate2 := &TestDelegateDispatcherAggregate{}
	s.disp.AddHandler(aggregate2, TestCommand{})
	c.Assert(len(s.disp.commandHandlers), Equals, 1)
	commandType := reflect.TypeOf(TestCommand{})
	c.Assert(s.disp.commandHandlers, t.HasKey, commandType)
	aggregateBaseType := reflect.ValueOf(aggregate).Elem().Type()
	c.Assert(s.disp.commandHandlers[commandType], Equals, aggregateBaseType)
}

type TestGlobalSubscriberDelegateDispatcher struct {
	handledEvent Event
}

func (t *TestGlobalSubscriberDelegateDispatcher) HandleEvent(event Event) {
	t.handledEvent = event
}

func (s *DelegateDispatcherSuite) Test_HandleCommand_Simple(c *C) {
	aggregate := &TestDelegateDispatcherAggregate{}
	s.disp.AddHandler(aggregate, TestCommand{})
	command1 := TestCommand{NewUUID(), "command1"}
	err := s.disp.Dispatch(command1)
	c.Assert(err, Equals, nil)
	c.Assert(dispatchedDelegateCommand, Equals, command1)
	c.Assert(len(s.store.events), Equals, 1)
	c.Assert(s.store.events[0], DeepEquals, TestEvent{command1.TestID, command1.Content})
	c.Assert(len(s.bus.events), Equals, 1)
	c.Assert(s.bus.events[0], DeepEquals, TestEvent{command1.TestID, command1.Content})
}

func (s *DelegateDispatcherSuite) Test_HandleCommand_ErrorInHandler(c *C) {
	aggregate := &TestDelegateDispatcherAggregate{}
	s.disp.AddHandler(aggregate, TestCommand{})
	commandError := TestCommand{NewUUID(), "error"}
	err := s.disp.Dispatch(commandError)
	c.Assert(dispatchedDelegateCommand, Equals, commandError)
	c.Assert(err, ErrorMatches, "command error")
	c.Assert(len(s.store.events), Equals, 0)
	c.Assert(len(s.bus.events), Equals, 0)
}

var callCountDelegateDispatcher int

type BenchmarkDelegateDispatcherAggregate struct {
	Aggregate
}

func (t *BenchmarkDelegateDispatcherAggregate) HandleCommand(command Command) ([]Event, error) {
	callCountDelegateDispatcher++
	return nil, nil
}

func (t *BenchmarkDelegateDispatcherAggregate) HandleEvent(event Event) {
}

func (s *DelegateDispatcherSuite) Benchmark_DelegateDispatcher(c *C) {
	store := &MockEventStore{
		events: make([]Event, 0),
	}
	bus := &MockEventBus{
		events: make([]Event, 0),
	}
	disp := NewDelegateDispatcher(store, bus)
	agg := &BenchmarkDelegateDispatcherAggregate{}
	disp.AddHandler(agg, TestCommand{})

	callCountDelegateDispatcher = 0
	command1 := TestCommand{NewUUID(), "command1"}
	for i := 0; i < c.N; i++ {
		disp.Dispatch(command1)
	}
	c.Assert(callCountDelegateDispatcher, Equals, c.N)
}

type ReflectDispatcherSuite struct {
	store *MockEventStore
	bus   *MockEventBus
	disp  *ReflectDispatcher
}

func (s *ReflectDispatcherSuite) SetUpTest(c *C) {
	s.store = &MockEventStore{
		events: make([]Event, 0),
	}
	s.bus = &MockEventBus{
		events: make([]Event, 0),
	}
	s.disp = NewReflectDispatcher(s.store, s.bus)
}

func (s *ReflectDispatcherSuite) Test_NewReflectAggregate(c *C) {
	store := &MockEventStore{
		events: make([]Event, 0),
	}
	bus := &MockEventBus{
		events: make([]Event, 0),
	}
	disp := NewReflectDispatcher(store, bus)
	c.Assert(disp, Not(Equals), nil)
	c.Assert(disp.eventStore, Equals, store)
	c.Assert(disp.eventBus, Equals, bus)
	c.Assert(disp.commandHandlers, Not(Equals), nil)
}

var dispatchedCommand Command

type TestSource struct {
	Aggregate
}

func (t *TestSource) HandleTestCommand(command TestCommand) ([]Event, error) {
	dispatchedCommand = command
	if command.Content == "error" {
		return nil, fmt.Errorf("command error")
	}
	return []Event{TestEvent{command.TestID, command.Content}}, nil
}

func (t *TestSource) HandleCommandOther2(command TestCommandOther2, invalidParam string) ([]Event, error) {
	return nil, nil
}

func (s *ReflectDispatcherSuite) Test_Dispatch_Simple(c *C) {
	dispatchedCommand = nil
	source := &TestSource{}
	sourceType := reflect.ValueOf(source).Elem().Type()
	method, _ := reflect.TypeOf(source).MethodByName("HandleTestCommand")
	s.disp.commandHandlers[reflect.TypeOf(TestCommand{})] = handler{
		sourceType: sourceType,
		method:     method,
	}
	command1 := TestCommand{NewUUID(), "command1"}
	err := s.disp.Dispatch(command1)
	c.Assert(dispatchedCommand, Equals, command1)
	c.Assert(err, Equals, nil)
}

func (s *ReflectDispatcherSuite) Test_Dispatch_MissingField(c *C) {
	dispatchedCommand = nil
	source := &TestSource{}
	sourceType := reflect.ValueOf(source).Elem().Type()
	method, _ := reflect.TypeOf(source).MethodByName("HandleTestCommand")
	s.disp.commandHandlers[reflect.TypeOf(TestCommand{})] = handler{
		sourceType: sourceType,
		method:     method,
	}
	command1 := TestCommand{Content: "command1"}
	err := s.disp.Dispatch(command1)
	c.Assert(err, ErrorMatches, "missing field: TestID")
	c.Assert(dispatchedCommand, Equals, nil)
}

func (s *ReflectDispatcherSuite) Test_Dispatch_ErrorInHandler(c *C) {
	dispatchedCommand = nil
	source := &TestSource{}
	sourceType := reflect.ValueOf(source).Elem().Type()
	method, _ := reflect.TypeOf(source).MethodByName("HandleTestCommand")
	s.disp.commandHandlers[reflect.TypeOf(TestCommand{})] = handler{
		sourceType: sourceType,
		method:     method,
	}
	commandError := TestCommand{NewUUID(), "error"}
	err := s.disp.Dispatch(commandError)
	c.Assert(err, ErrorMatches, "command error")
	c.Assert(dispatchedCommand, Equals, commandError)
}

func (s *ReflectDispatcherSuite) Test_Dispatch_NoHandlers(c *C) {
	command1 := TestCommand{NewUUID(), "command1"}
	err := s.disp.Dispatch(command1)
	c.Assert(err, ErrorMatches, "no handlers for command")
}

func (s *ReflectDispatcherSuite) Test_AddHandler_Simple(c *C) {
	source := &TestSource{}
	s.disp.AddHandler(source, TestCommand{})
	c.Assert(len(s.disp.commandHandlers), Equals, 1)
	commandType := reflect.TypeOf(TestCommand{})
	c.Assert(s.disp.commandHandlers, t.HasKey, commandType)
	sourceType := reflect.ValueOf(source).Elem().Type()
	method, _ := reflect.TypeOf(source).MethodByName("HandleTestCommand")
	sourceHandler := handler{
		sourceType: sourceType,
		method:     method,
	}
	c.Assert(s.disp.commandHandlers[commandType], Equals, sourceHandler)
}

func (s *ReflectDispatcherSuite) Test_AddHandler_Duplicate(c *C) {
	source := &TestSource{}
	s.disp.AddHandler(source, TestCommand{})
	source2 := &TestSource{}
	s.disp.AddHandler(source2, TestCommand{})
	c.Assert(len(s.disp.commandHandlers), Equals, 1)
	commandType := reflect.TypeOf(TestCommand{})
	c.Assert(s.disp.commandHandlers, t.HasKey, commandType)
	sourceType := reflect.ValueOf(source).Elem().Type()
	method, _ := reflect.TypeOf(source).MethodByName("HandleTestCommand")
	sourceHandler := handler{
		sourceType: sourceType,
		method:     method,
	}
	c.Assert(s.disp.commandHandlers[commandType], Equals, sourceHandler)
}

func (s *ReflectDispatcherSuite) Test_AddHandler_MissingMethod(c *C) {
	source := &TestSource{}
	s.disp.AddHandler(source, TestCommandOther{})
	c.Assert(len(s.disp.commandHandlers), Equals, 0)
}

func (s *ReflectDispatcherSuite) Test_AddHandler_IncorrectMethod(c *C) {
	source := &TestSource{}
	s.disp.AddHandler(source, TestCommandOther2{})
	c.Assert(len(s.disp.commandHandlers), Equals, 0)
}

type TestGlobalSubscriber struct {
	handledEvent Event
}

func (t *TestGlobalSubscriber) HandleEvent(event Event) {
	t.handledEvent = event
}

func (s *ReflectDispatcherSuite) Test_HandleCommand_Simple(c *C) {
	dispatchedCommand = nil
	source := &TestSource{}
	s.disp.AddHandler(source, TestCommand{})
	command1 := TestCommand{NewUUID(), "command1"}
	err := s.disp.Dispatch(command1)
	c.Assert(err, Equals, nil)
	c.Assert(dispatchedCommand, Equals, command1)
	c.Assert(len(s.store.events), Equals, 1)
	c.Assert(s.store.events[0], DeepEquals, TestEvent{command1.TestID, command1.Content})
	c.Assert(len(s.bus.events), Equals, 1)
	c.Assert(s.bus.events[0], DeepEquals, TestEvent{command1.TestID, command1.Content})
}

func (s *ReflectDispatcherSuite) Test_HandleCommand_MissingField(c *C) {
	dispatchedCommand = nil
	source := &TestSource{}
	s.disp.AddHandler(source, TestCommand{})
	command1 := TestCommand{Content: "command1"}
	err := s.disp.Dispatch(command1)
	c.Assert(err, ErrorMatches, "missing field: TestID")
	c.Assert(dispatchedCommand, Equals, nil)
	c.Assert(len(s.store.events), Equals, 0)
	c.Assert(len(s.bus.events), Equals, 0)
}

func (s *ReflectDispatcherSuite) Test_HandleCommand_ErrorInHandler(c *C) {
	dispatchedCommand = nil
	source := &TestSource{}
	s.disp.AddHandler(source, TestCommand{})
	commandError := TestCommand{NewUUID(), "error"}
	err := s.disp.Dispatch(commandError)
	c.Assert(err, ErrorMatches, "command error")
	c.Assert(dispatchedCommand, Equals, commandError)
	c.Assert(len(s.store.events), Equals, 0)
	c.Assert(len(s.bus.events), Equals, 0)
}

var callCount int

type BenchmarkAggregate struct {
	Aggregate
}

func (t *BenchmarkAggregate) HandleTestCommand(command TestCommand) ([]Event, error) {
	callCount++
	return nil, nil
}

func (s *ReflectDispatcherSuite) Benchmark_ReflectDispatcher(c *C) {
	store := &MockEventStore{
		events: make([]Event, 0),
	}
	bus := &MockEventBus{
		events: make([]Event, 0),
	}
	disp := NewReflectDispatcher(store, bus)
	agg := &BenchmarkAggregate{}
	disp.AddHandler(agg, TestCommand{})

	callCount = 0
	command1 := TestCommand{NewUUID(), "command1"}
	for i := 0; i < c.N; i++ {
		disp.Dispatch(command1)
	}
	c.Assert(callCount, Equals, c.N)
}

type DispatcherSuite struct{}

func (s *DispatcherSuite) Test_CheckCommand_AllFields(c *C) {
	command := TestCommand{NewUUID(), "command1"}
	err := checkCommand(command)
	c.Assert(err, Equals, nil)
}

type TestCommandValue struct {
	TestID  UUID
	Content string
}

func (t TestCommandValue) AggregateID() UUID { return t.TestID }

func (s *DispatcherSuite) Test_CheckCommand_MissingRequired_Value(c *C) {
	command := TestCommandValue{TestID: NewUUID()}
	err := checkCommand(command)
	c.Assert(err, ErrorMatches, "missing field: Content")
}

type TestCommandSlice struct {
	TestID UUID
	Slice  []string
}

func (t TestCommandSlice) AggregateID() UUID { return t.TestID }

func (s *DispatcherSuite) Test_CheckCommand_MissingRequired_Slice(c *C) {
	command := TestCommandSlice{TestID: NewUUID()}
	err := checkCommand(command)
	c.Assert(err, ErrorMatches, "missing field: Slice")
}

type TestCommandMap struct {
	TestID UUID
	Map    map[string]string
}

func (t TestCommandMap) AggregateID() UUID { return t.TestID }

func (s *DispatcherSuite) Test_CheckCommand_MissingRequired_Map(c *C) {
	command := TestCommandMap{TestID: NewUUID()}
	err := checkCommand(command)
	c.Assert(err, ErrorMatches, "missing field: Map")
}

type TestCommandStruct struct {
	TestID UUID
	Struct struct {
		Test string
	}
}

func (t TestCommandStruct) AggregateID() UUID { return t.TestID }

func (s *DispatcherSuite) Test_CheckCommand_MissingRequired_Struct(c *C) {
	command := TestCommandStruct{TestID: NewUUID()}
	err := checkCommand(command)
	c.Assert(err, ErrorMatches, "missing field: Struct")
}

type TestCommandTime struct {
	TestID UUID
	Time   time.Time
}

func (t TestCommandTime) AggregateID() UUID { return t.TestID }

func (s *DispatcherSuite) Test_CheckCommand_MissingRequired_Time(c *C) {
	command := TestCommandTime{TestID: NewUUID()}
	err := checkCommand(command)
	c.Assert(err, ErrorMatches, "missing field: Time")
}

type TestCommandOptional struct {
	TestID  UUID
	Content string `eh:"optional"`
}

func (t TestCommandOptional) AggregateID() UUID { return t.TestID }

func (s *DispatcherSuite) Test_CheckCommand_MissingOptionalField(c *C) {
	command := TestCommandOptional{TestID: NewUUID()}
	err := checkCommand(command)
	c.Assert(err, Equals, nil)
}

type TestCommandPrivate struct {
	TestID  UUID
	private string
}

func (t TestCommandPrivate) AggregateID() UUID { return t.TestID }

func (s *DispatcherSuite) Test_CheckCommand_MissingPrivateField(c *C) {
	command := TestCommandPrivate{TestID: NewUUID()}
	err := checkCommand(command)
	c.Assert(err, Equals, nil)
}
