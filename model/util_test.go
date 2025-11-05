package model

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type (
	t1 struct{ Empty }
	t2 struct{}
	t3 struct{ Name string }
	t4 struct {
		Name string
		Empty
	}
)

func TestAreTypesEqual(t *testing.T) {
	require.True(t, AreTypesEqual[*User, *User, *User]())
	require.False(t, AreTypesEqual[*User, User, *User]())
	require.False(t, AreTypesEqual[*User, *User, User]())
	require.False(t, AreTypesEqual[*User, User, User]())
	require.False(t, AreTypesEqual[*User, *Menu, *Menu]())
	require.False(t, AreTypesEqual[*User, string, *User]())
	require.False(t, AreTypesEqual[*User, *User, int]())
	require.False(t, AreTypesEqual[t1, t1, t1]())
	require.True(t, AreTypesEqual[t4, t4, t4]())
	require.False(t, AreTypesEqual[t1, *User, User]())
	require.False(t, AreTypesEqual[t1, int, *string]())
}

func BenchmarkAreTypesEqual(b *testing.B) {
	b.Run("test1", func(b *testing.B) {
		for b.Loop() {
			AreTypesEqual[*User, *User, *User]()
		}
	})
	b.Run("test2", func(b *testing.B) {
		for b.Loop() {
			AreTypesEqual[*User, User, *User]()
		}
	})
	b.Run("test3", func(b *testing.B) {
		for b.Loop() {
			AreTypesEqual[*User, *User, User]()
		}
	})
	b.Run("test4", func(b *testing.B) {
		for b.Loop() {
			AreTypesEqual[*User, User, User]()
		}
	})
	b.Run("test5", func(b *testing.B) {
		for b.Loop() {
			AreTypesEqual[*User, *Menu, *Menu]()
		}
	})
	b.Run("test6", func(b *testing.B) {
		for b.Loop() {
			AreTypesEqual[*User, string, *User]()
		}
	})
	b.Run("test7", func(b *testing.B) {
		for b.Loop() {
			AreTypesEqual[*User, *User, int]()
		}
	})
}

func TestIsModelEmpty(t *testing.T) {
	require.True(t, IsModelEmpty[t1]())
	require.True(t, IsModelEmpty[t2]())
	require.False(t, IsModelEmpty[t3]())
	require.False(t, IsModelEmpty[t4]())
}

func TestIsValid(t *testing.T) {
	type t1 string
	type t2 int
	type t3 struct{}
	type t4 struct{ Empty }
	type t5 struct{ Any }
	type t6 struct{ Base }

	require.False(t, IsValid[t1]())
	require.False(t, IsValid[*t1]())
	require.False(t, IsValid[t2]())
	require.False(t, IsValid[*t2]())
	require.False(t, IsValid[t3]())
	require.False(t, IsValid[*t3]())
	require.False(t, IsValid[t4]())
	require.False(t, IsValid[*t4]())
	require.False(t, IsValid[t5]())
	require.False(t, IsValid[*t5]())
	require.False(t, IsValid[t6]())
	require.True(t, IsValid[*t6]())
}

func BenchmarkIsModelEmpty(b *testing.B) {
	b.Run("test", func(b *testing.B) {
		for b.Loop() {
			_ = IsModelEmpty[t1]()
		}
	})
}
