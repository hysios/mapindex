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
		p    reflect.Value // 父容器
		c    reflect.Value // 容器
		pkey reflect.Value
		pidx int
		opt  Option
	)

	setOpts(&opt, opts)
	c = v
	for i, s := range ss {
		idx, num := convInt(s)

		// 最后值
		if i == l-1 {
			switch c.Kind() {
			case reflect.Map:
				c.SetMapIndex(reflect.ValueOf(s), val)
				return nil
			case reflect.Slice:
				if !num {
					pkey = reflect.ValueOf(s)
					cv := reflect.MakeMap(mapTyp)
					// p.Index(pidx)
					cv.SetMapIndex(pkey, val)
					cm := c.Index(pidx)
					cm.Set(cv)
					// c.Set(cv)
					return nil
					// return errors.New("not a index in slice")
				}

				if idx < 0 {
					switch p.Kind() {
					case reflect.Map:
						p.SetMapIndex(pkey, reflect.Append(c, val))
					case reflect.Slice:
						cv := p.Index(pidx)
						cv.Set(reflect.Append(c, val))
					default:
						return errors.New("invalid set type")
					}

					return nil
				} else if idx > 0 && idx < c.Len() {
					cv := c.Index(idx)
					cv.Set(val)
					return nil
				} else {
					if opt.AutoExpand && idx < opt.SliceMaxLimit {
						var cv reflect.Value
						cv = reflect.MakeSlice(sliceTyp, idx+1, opt.SliceMaxLimit+1)
						reflect.Copy(cv, c)
						mvv := cv.Index(idx)
						mvv.Set(val)

						switch p.Kind() {
						case reflect.Map:
							p.SetMapIndex(pkey, cv)
						case reflect.Slice:
							cv := p.Index(pidx)
							cv.Set(cv)
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
		switch c.Kind() {
		case reflect.Map:
			p = c
			pkey = reflect.ValueOf(s)
			cm := c.MapIndex(pkey)
			if !cm.IsValid() {
				var cv reflect.Value
				if _, nnum := convInt(ss[i+1]); nnum {
					cv = reflect.MakeSlice(sliceTyp, 0, 0)
				} else {
					cv = reflect.MakeMap(mapTyp)
				}
				c.SetMapIndex(pkey, cv)
				c = cv
			} else {
				c = cm.Elem()
				log.Printf("v kind %s", c.Kind())
			}
		case reflect.Slice, reflect.Array:
			if !num {
				return errors.New("not a index in slice")
			}
			if idx >= 0 && idx < c.Len() {
				p = c
				pidx = i
				c = c.Index(idx).Elem()
				log.Printf("v kind %s", c.Kind())
			} else if idx >= 0 {
				if opt.AutoExpand && idx < opt.SliceMaxLimit {
					var cv reflect.Value
					cv = reflect.MakeSlice(sliceTyp, idx+1, opt.SliceMaxLimit)
					reflect.Copy(cv, c)

					switch p.Kind() {
					case reflect.Map:
						p.SetMapIndex(pkey, cv)
					case reflect.Slice:
						pcv := p.Index(pidx)
						pcv.Set(cv)
					default:
						return errors.New("invalid set type")
					}
					pidx = idx
					p = c
					c = cv
				} else if opt.AutoExpand {
					return errors.New("out of autoexpend max slice limit")
				} else {
					return errors.New("index out of slice length")
				}
			} else {
				var cv reflect.Value
				if _, nnum := convInt(ss[i+1]); nnum {
					cv = reflect.MakeSlice(sliceTyp, 0, 0)
				} else {
					cv = reflect.MakeMap(mapTyp)
				}
				log.Println("cv", cv, c.Type().Elem())
				switch p.Kind() {
				case reflect.Map:
					p.SetMapIndex(pkey, reflect.Append(c, cv))
				case reflect.Slice:
					pcv := p.Index(pidx)
					pcv.Set(reflect.Append(c, cv))
				default:
					return errors.New("invalid set type")
				}
				p = c
				c = cv
			}
		default:
			return errors.New("must a slice or map")
		}
	}

	return nil
}

var Nil = reflect.New(interTyp)

func elemval(m reflect.Value) reflect.Value {
	if m.IsValid() {
		return m.Elem()
	}
	return m
}

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
				return elemval(m.MapIndex(pkey)), true
			case reflect.Slice, reflect.Array:
				if !num {
					log.Printf("invalid index type of slice or array")
					return Nil, false
				}

				if idx >= 0 && idx < m.Len() {
					return elemval(m.Index(idx)), true
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
		if val.IsValid() {
			return val.Interface()
		} else {
			return val
		}
	}
	return nil
}
