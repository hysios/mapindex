package mapindex

import (
	"log"
	"reflect"
	"testing"

	"github.com/kr/pretty"
	"github.com/stretchr/testify/assert"
	"github.com/yudai/pp"
)

func Test_getIndexPath(t *testing.T) {
	var m = map[string]interface{}{
		"name":    map[string]interface{}{"first": "Tom", "last": "Smith"},
		"friends": []interface{}{"bob", "tom"},
	}

	v := reflect.ValueOf(&m)
	vv, ok := getIndexPath(v, "name")
	assert.Equal(t, ok, true)
	log.Print(vv)

	vv, ok = getIndexPath(vv, "last")
	assert.Equal(t, ok, true)
	assert.Equal(t, vv.Interface(), "Smith")
	log.Print(vv)

	v = reflect.ValueOf(&m)
	vv, ok = getIndexPath(v, "friends")
	assert.Equal(t, ok, true)
	assert.Contains(t, vv.Interface(), "bob")
	assert.Contains(t, vv.Interface(), "tom")
	log.Print(vv)

	vv, ok = getIndexPath(vv, "1")
	assert.Equal(t, ok, true)
	assert.Equal(t, vv.Interface(), "tom")
	log.Print(vv)

	vv, ok = getIndexPath(v, "test")
	assert.False(t, ok)
	assert.Zero(t, vv)
	log.Print(vv)
}

func Test_getIndexPathSlice(t *testing.T) {
	var m = map[string]interface{}{
		"name": map[string]interface{}{
			"first": "Tom",
			"last":  "Smith",
		},
		"company": map[string]interface{}{
			"name": "pdls",
			"locations": []interface{}{
				map[string]interface{}{
					"name":    "headquarter",
					"default": true,
					"road1":   "麓谷企业广场",
					"members": []interface{}{
						map[string]interface{}{
							"username": "elle",
							"salary":   1000,
							"years":    3,
						},
						map[string]interface{}{
							"username": "jon",
							"salary":   1500.0,
							"years":    2,
						},
					},
				},
				map[string]interface{}{
					"name":    "subpart",
					"default": false,
					"road1":   "河东",
				},
			},
		},
		"friends": []interface{}{"bob", "tom"},
	}

	v := reflect.ValueOf(&m)
	vv, ok := getIndexPath(v, "friends.1")
	assert.Equal(t, ok, true)
	assert.Equal(t, vv.Interface(), "tom")

	v = reflect.ValueOf(&m)
	vv, ok = getIndexPath(v, "company.locations.0.name")
	assert.Equal(t, ok, true)
	assert.Equal(t, vv.Interface(), "headquarter")

}

func assertMapValue(t *testing.T, contains, m map[string]interface{}) bool {
	for key, val := range m {
		if v, ok := contains[key]; ok {
			switch mv := v.(type) {
			case map[string]interface{}:
				assertMapValue(t, mv, val.(map[string]interface{}))
			default:
				if mv == val {
					return true
				}

				t.Fatalf("not continas key: %s => val: %v", key, val)
				return false
			}
		}
	}

	return false
}

func Test_setIndexPath(t *testing.T) {
	var m = map[string]interface{}{
		"name":    map[string]interface{}{"first": "Tom", "last": "Smith"},
		"friends": []interface{}{"bob", "tom"},
	}
	v := reflect.ValueOf(&m)

	err := setIndexPath(v, "name.last", reflect.ValueOf("bob"))
	assert.NoError(t, err)
	// assert.Equal(t, MapGetPath(m, "name.last"), "bob")

	err = setIndexPath(v, "friends.1", reflect.ValueOf("jack"))
	assert.NoError(t, err)

	err = setIndexPath(v, "friends.-1", reflect.ValueOf("fred"))
	assert.NoError(t, err)
	assert.Len(t, m["friends"], 3)

	err = setIndexPath(v, "friends.-1", reflect.ValueOf("scarlet"), OptOverwrite())
	assert.NoError(t, err)
	assert.Len(t, m["friends"], 4)

	err = setIndexPath(v, "friends.test.key", reflect.ValueOf("scarlet"), OptOverwrite())
	assert.NoError(t, err)
	assertMapValue(t, m, map[string]interface{}{"test": map[string]interface{}{"key": "scarlet"}})

	err = setIndexPath(v, "friends", reflect.ValueOf("scarlet"), OptOverwrite())
	assert.NoError(t, err)
	assert.Equal(t, m["friends"], "scarlet")

	err = setIndexPath(v, "name.object.age", reflect.ValueOf(1), OptOverwrite())
}

func Test_setIndexComplex(t *testing.T) {
	var m = map[string]interface{}{
		"name": map[string]interface{}{
			"first": "Tom",
			"last":  "Smith",
		},
		"company": map[string]interface{}{
			"name": "pdls",
			"locations": []interface{}{
				map[string]interface{}{
					"name":    "headquarter",
					"default": true,
					"road1":   "麓谷企业广场",
					"members": []interface{}{
						map[string]interface{}{
							"username": "elle",
							"salary":   1000,
							"years":    3,
						},
						map[string]interface{}{
							"username": "jon",
							"salary":   1500.0,
							"years":    2,
						},
					},
				},
				map[string]interface{}{
					"name":    "subpart",
					"default": false,
					"road1":   "河东",
				},
			},
		},
		"friends": []interface{}{"bob", "tom"},
	}

	err := Set(m, "company.name", "bob")
	assert.NoError(t, err)
	assert.Equal(t, Get(m, "company.name"), "bob")

	v := reflect.ValueOf(&m)
	err = Set(m, "company.locations[0].default", false)
	assert.NoError(t, err)
	// assert.Equal(t, Get(m, "company.locations[0].default"), false)
	print(t, m)

	// add field
	err = setIndexPath(v, "company.locations.0.road2", reflect.ValueOf("C1 栋 702"))
	assert.NoError(t, err)
	print(t, m)

	err = setIndexPath(v, "company.locations.-1.name", reflect.ValueOf("C1 栋 702"))
	assert.NoError(t, err)
	print(t, m)

	err = setIndexPath(v, "company.locations.-1.name", reflect.ValueOf("C1 栋 702"))
	assert.NoError(t, err)
	print(t, m)

	err = setIndexPath(v, "company.locations.0.medmbers.-1.username", reflect.ValueOf("zhang"))
	assert.NoError(t, err)

	err = setIndexPath(v, "company.locations.0.memmbers.username", reflect.ValueOf("zhang"))
	assert.NoError(t, err)
	print(t, m)

	err = setIndexPath(v, "company.name.memmbers", reflect.ValueOf("zhang"), OptOverwrite())
	assert.NoError(t, err)
	print(t, m)

	err = setIndexPath(v, "company.name.0", reflect.ValueOf("zhang"), OptOverwrite())
	assert.NoError(t, err)
	print(t, m)

	err = setIndexPath(v, "friends.-1", reflect.ValueOf("zhang"), OptOverwrite())
	assert.NoError(t, err)
	print(t, m)
}

func Test_setIndexComplexOutOfSliceIndex(t *testing.T) {
	var m = map[string]interface{}{
		"name":    map[string]interface{}{"first": "Tom", "last": "Smith"},
		"friends": []interface{}{"bob", "tom"},
	}
	v := reflect.ValueOf(&m)

	err := setIndexPath(v, "friends.4", reflect.ValueOf("bob"))
	assert.Error(t, err, "index out of slice length")
	pp.Print(m)

	err = setIndexPath(v, "friends.4", reflect.ValueOf("bob"), OptSliceMax(1024))
	assert.NoError(t, err)
	pp.Print(m)

	err = setIndexPath(v, "friends.5.name", reflect.ValueOf("jim"), OptSliceMax(1024))
	assert.NoError(t, err)
	pp.Print(m)
}

func TestGet(t *testing.T) {
	var m = map[string]interface{}{
		"name": map[string]interface{}{
			"first": "Tom",
			"last":  "Smith",
		},
		"company": map[string]interface{}{
			"name": "pdls",
			"locations": []interface{}{
				map[string]interface{}{
					"name":    "headquarter",
					"default": true,
					"road1":   "麓谷企业广场",
					"members": []interface{}{
						map[string]interface{}{
							"username": "elle",
							"salary":   1000,
							"years":    3,
						},
						map[string]interface{}{
							"username": "jon",
							"salary":   1500.0,
							"years":    2,
						},
					},
				},
				map[string]interface{}{
					"name":    "subpart",
					"default": false,
					"road1":   "河东",
				},
			},
		},
		"friends": []interface{}{"bob", "tom"},
	}
	type args struct {
		m        interface{}
		selector string
	}
	tests := []struct {
		name string
		args args
		want interface{}
	}{
		{
			args: args{
				m:        m,
				selector: "name",
			},
			want: map[string]interface{}{
				"first": "Tom",
				"last":  "Smith",
			},
		},

		{
			args: args{
				m:        m,
				selector: "company.locations.0.name",
			},
			want: "headquarter",
		},

		{
			args: args{
				m:        m,
				selector: "company.locations.1.name",
			},
			want: "subpart",
		},

		{
			args: args{
				m:        m,
				selector: "company.locations.1.default",
			},
			want: false,
		},
		{
			args: args{
				m:        m,
				selector: "company.locations.3.default",
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Get(tt.args.m, tt.args.selector); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Get() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_setIndexPathEmpty(t *testing.T) {
	var m = map[string]interface{}{}
	v := reflect.ValueOf(&m)

	err := setIndexPath(v, "platePic.mediaType", reflect.ValueOf("image/jpeg"))
	assert.NoError(t, err)
	pp.Println(m)

	err = setIndexPath(v, "platePic.size", reflect.ValueOf(123456))
	assert.NoError(t, err)
	pp.Println(m)

	err = setIndexPath(v, "platePic.data", reflect.ValueOf("hello world"))
	assert.NoError(t, err)
	pp.Println(m)
}

func Test_Set(t *testing.T) {
	var m = map[string]interface{}{}
	Set(m, "a.b.c", true)
	print(t, m)

	Set(m, "a.b", true)
	print(t, m)

	Set(m, "a.e[1]", 123)
	print(t, m)

	Set(m, "a.e[3]", false)
	print(t, m)

	Set(m, "a.e[4].c", true)
	print(t, m)
}

func Test_deepSearch(t *testing.T) {
	var (
		m = map[string]interface{}{}
		c = deepSearch(m, nil, nil, []string{"a", "b", "c"})
	)
	if v, ok := c.(map[string]interface{}); ok {
		v["d"] = true
	}
	t.Log(m)

	c = deepSearch(m, nil, nil, []string{"a", "b"})
	if v, ok := c.(map[string]interface{}); ok {
		v["e"] = true
	}

	t.Log(m)

	c = deepSearch(m, nil, nil, []string{"a"})
	if v, ok := c.(map[string]interface{}); ok {
		v["f"] = true
	}

	t.Log(m)

	c = deepSearch(m, nil, nil, []string{"a", "e[1]"})
	if v, ok := c.([]interface{}); ok {
		v[1] = true
		_ = v
	}

	t.Log(m)

	c = deepSearch(m, nil, nil, []string{"a", "e[4]"})
	if v, ok := c.([]interface{}); ok {
		v[4] = false
	}

	t.Log(m)

	c = deepSearch(m, nil, nil, []string{"a", "e[3]", "c"})
	if v, ok := c.(map[string]interface{}); ok {
		v["d"] = 1
	}
	t.Log(m)

	c = deepSearch(m, nil, nil, []string{"b", "e[2]", "c"})
	if v, ok := c.(map[string]interface{}); ok {
		v["d"] = 1
	}
	print(t, m)

	c = deepSearch(m, nil, nil, []string{"b", "e", "c"})
	if v, ok := c.(map[string]interface{}); ok {
		v["d"] = 2
	}
	print(t, m)

	c = deepSearch(m, nil, nil, []string{"b", "e", "c[1]"})
	if v, ok := c.([]interface{}); ok {
		v[1] = 123
	}
	print(t, m)

	// c = deepSearch(m, nil, nil, []string{"b", "e", "c[1][2]"})
	// if v, ok := c.([]interface{}); ok {
	// 	v[1] = 123
	// }
	// print(t, m)
}

func print(t *testing.T, val interface{}) {
	t.Logf("% #v", pretty.Formatter(val))
}
