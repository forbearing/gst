package model

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type (
	t1 struct{ *Empty }
	t2 struct{}
	t3 struct{ Name string }
	t4 struct {
		Name string
		*Empty
	}
)

func TestAreTypesEqual(t *testing.T) {
	require.True(t, AreTypesEqual[*User, *User, *User]())
	require.False(t, AreTypesEqual[*User, User, *User]())
	require.False(t, AreTypesEqual[*User, *User, User]())
	require.False(t, AreTypesEqual[*User, User, User]())
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

func TestIsEmpty(t *testing.T) {
	type t1 string
	type t2 int
	type t3 struct{}
	type t4 struct{ Empty }
	type t5 struct{ *Empty }
	type t6 struct{ Any }
	type t7 struct{ *Any }
	type t8 struct{ Empty Any }
	type t9 struct {
		*Empty
		Any
	}
	type t10 struct {
		Empty
		*Any
	}
	type t11 struct {
		Empty
		*Any
	}
	type t12 struct {
		a string
	}
	type t13 struct {
		a string
		Empty
	}
	type t14 struct {
		a string
		Any
	}
	type t15 = Empty
	type t16 = Any

	require.True(t, IsEmpty[t1]())
	require.True(t, IsEmpty[t2]())
	require.True(t, IsEmpty[t3]())
	require.True(t, IsEmpty[t4]())
	require.True(t, IsEmpty[t5]())
	require.True(t, IsEmpty[t6]())
	require.True(t, IsEmpty[t7]())
	require.True(t, IsEmpty[t8]())
	require.True(t, IsEmpty[t9]())
	require.True(t, IsEmpty[t10]())
	require.True(t, IsEmpty[t11]())
	require.False(t, IsEmpty[t12]())
	require.False(t, IsEmpty[t13]())
	require.False(t, IsEmpty[t14]())
	require.True(t, IsEmpty[t15]())
	require.True(t, IsEmpty[*t15]())
	require.True(t, IsEmpty[t16]())
	require.True(t, IsEmpty[*t16]())
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
			_ = IsEmpty[t1]()
		}
	})
}
