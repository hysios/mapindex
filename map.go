package mapindex

import (
	"errors"
	"log"
	"reflect"
	"strconv"
	"strings"
)

type Option struct {
	SliceMaxLimit int
	AutoExpand    bool
}

type SetOptFunc func(opts *Option)

var (
	mapTyp   = reflect.TypeOf(make(map[string]interface{}))
	sliceTyp = reflect.TypeOf(make([]interface{}, 0))
	interTyp = reflect.TypeOf(new(interface{}))
)

func convInt(s string) (int, bool) {
	idx, err := strconv.Atoi(s)
	if err != nil {
		return 0, false
	} else {
		return idx, true
	}
}

func OptSliceMax(n int) SetOptFunc {
	return func(opts *Option) {
		if n > 0 {
			opts.AutoExpand = true
		}
		opts.SliceMaxLimit = n
	}
}

func setOpts(opts *Option, setOpts []SetOptFunc) {
	for _, set := range setOpts {
		set(opts)
	}

}

func setIndexPath(v reflect.Value, selector string, val reflect.Value, opts ...SetOptFunc) error {
	v = reflect.Indirect(v)
	var (
		ss   = strings.Split(selector, ".")
		l    = len(ss)
		p    reflect.Value
		m    reflect.Value
		pkey reflect.Value
		pidx int
		opt  Option
	)

	setOpts(&opt, opts)
	m = v
	for i, s := range ss {
		idx, num := convInt(s)

		// 最后值
		if i == l-1 {
			switch m.Kind() {
			case reflect.Map:
				m.SetMapIndex(reflect.ValueOf(s), val)
				return nil
			case reflect.Slice:
				if !num {
					return errors.New("not a index in slice")
				}

				if idx < 0 {
					switch p.Kind() {
					case reflect.Map:
						p.SetMapIndex(pkey, reflect.Append(m, val))
					case reflect.Slice:
						mv := p.Index(pidx)
						mv.Set(reflect.Append(m, val))
					default:
						return errors.New("invalid set type")
					}

					return nil
				} else if idx > 0 && idx < m.Len() {
					mv := m.Index(idx)
					mv.Set(val)
					return nil
				} else {
					if opt.AutoExpand && idx < opt.SliceMaxLimit {
						var mv reflect.Value
						mv = reflect.MakeSlice(sliceTyp, idx+1, opt.SliceMaxLimit+1)
						reflect.Copy(mv, m)
						mvv := mv.Index(idx)
						mvv.Set(val)

						switch p.Kind() {
						case reflect.Map:
							p.SetMapIndex(pkey, mv)
						case reflect.Slice:
							mv := p.Index(pidx)
							mv.Set(mv)
						default:
							return errors.New("invalid set type")
						}
						return nil
					} else if opt.AutoExpand {
						return errors.New("out of autoexpend max slice limit")
					}
				}
			default:
				return errors.New("must a slice or map")
			}
		}

		// 中间值
		switch m.Kind() {
		case reflect.Map:
			p = m
			pkey = reflect.ValueOf(s)
			mm := m.MapIndex(pkey)
			if !mm.IsValid() {
				var mv reflect.Value
				if _, nnum := convInt(ss[i+1]); nnum {
					mv = reflect.MakeSlice(sliceTyp, 0, 0)
				} else {
					mv = reflect.MakeMap(mapTyp)
				}
				m.SetMapIndex(pkey, mv)
				m = mv
			} else {
				m = mm.Elem()
				log.Printf("v kind %s", m.Kind())
			}
		case reflect.Slice, reflect.Array:
			if !num {
				return errors.New("not a index in slice")
			}
			if idx >= 0 && idx < m.Len() {
				p = m
				pidx = i
				m = m.Index(idx).Elem()
				log.Printf("v kind %s", m.Kind())
			} else if idx >= 0 {
				if opt.AutoExpand && idx < opt.SliceMaxLimit {
					var mv reflect.Value
					mv = reflect.MakeSlice(sliceTyp, idx+1, opt.SliceMaxLimit)
					reflect.Copy(mv, m)
					p = m
					m = mv
				} else if opt.AutoExpand {
					return errors.New("out of autoexpend max slice limit")
				}
				return errors.New("index out of slice length")
			} else {
				var mv reflect.Value
				if _, nnum := convInt(ss[i+1]); nnum {
					mv = reflect.MakeSlice(sliceTyp, 0, 0)
				} else {
					mv = reflect.MakeMap(mapTyp)
				}
				log.Println("mv", mv, m.Type().Elem())
				switch p.Kind() {
				case reflect.Map:
					p.SetMapIndex(pkey, reflect.Append(m, mv))
				case reflect.Slice:
					pmv := p.Index(pidx)
					pmv.Set(reflect.Append(m, mv))
				default:
					return errors.New("invalid set type")
				}
				p = m
				m = mv
			}
		default:
			return errors.New("must a slice or map")
		}
	}

	return nil
}

var Nil = reflect.New(interTyp)

func getIndexPath(v reflect.Value, selector string) (reflect.Value, bool) {
	v = reflect.Indirect(v)

	var (
		ss   = strings.Split(selector, ".")
		l    = len(ss)
		m    = v
		pkey reflect.Value
	)

	for i, s := range ss {
		idx, num := convInt(s)

		if i == l-1 {
			switch m.Kind() {
			case reflect.Map:
				pkey = reflect.ValueOf(s)
				return m.MapIndex(pkey).Elem(), true
			case reflect.Slice, reflect.Array:
				if !num {
					log.Printf("invalid index type of slice or array")
					return Nil, false
				}

				if idx >= 0 && idx < m.Len() {
					return m.Index(idx).Elem(), true
				} else if idx >= 0 {
					log.Printf("index out of slice length")
					return Nil, false
				} else {
					log.Printf("get index can't be negative")
					return Nil, false
				}
			default:
				return Nil, false
			}
		}

		switch m.Kind() {
		case reflect.Map:
			pkey = reflect.ValueOf(s)
			mm := m.MapIndex(pkey)
			m = mm.Elem()
		case reflect.Slice, reflect.Array:
			if !num {
				log.Printf("invalid index type of slice or array")
				return Nil, false
			}

			if idx >= 0 && idx < m.Len() {
				m = m.Index(idx).Elem()
			} else if idx >= 0 {
				log.Printf("index out of slice length")
				return Nil, false
			} else {
				log.Printf("get index can't be negative")
				return Nil, false
			}
		default:
			return Nil, false
		}
	}
	return Nil, false
}

func Set(m interface{}, selector string, val interface{}, opts ...SetOptFunc) error {
	v := reflect.ValueOf(m)
	return setIndexPath(v, selector, reflect.ValueOf(val), opts...)
}

func Get(m interface{}, selector string) interface{} {
	v := reflect.ValueOf(m)
	if val, ok := getIndexPath(v, selector); ok {

		return val.Interface()
	}
	return nil
}
