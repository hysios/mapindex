package mapindex

import (
	"errors"
	"log"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

type Option struct {
	SliceMaxLimit int
	AutoExpand    bool
	Overwrite     bool
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

func OptOverwrite() SetOptFunc {
	return func(opts *Option) {
		opts.Overwrite = true
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
				if !opt.Overwrite {
					return errors.New("must a slice or map")
				}

				c = reflect.MakeMap(mapTyp)
				p.SetMapIndex(pkey, c)
				c.SetMapIndex(reflect.ValueOf(s), val)
				return nil
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
				c = c.MapIndex(pkey).Elem()
				// c = cv
			} else {
				c = cm.Elem()
			}
		case reflect.Slice, reflect.Array:
			if !num {
				if !opt.Overwrite {
					return errors.New("not a index in slice")
				}
				c = reflect.MakeMap(mapTyp)
				p.SetMapIndex(pkey, c)
				c.SetMapIndex(reflect.ValueOf(s), val)
				return nil
			}
			if idx >= 0 && idx < c.Len() {
				p = c
				pidx = i
				c = c.Index(idx).Elem()
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

func searchMap(source map[string]interface{}, path []string) interface{} {
	if len(path) == 0 {
		return source
	}

	var (
		k, idxs, isSlice = isIndex(path[0])
	)

	if !isSlice {
		k = path[0]
	}

	next, ok := source[k]
	if ok {
		// Fast path
		if !isSlice && len(path) == 1 {
			return next
		}

		// Nested case
		switch x := next.(type) {
		// case map[interface{}]interface{}:
		// 	return searchMap(ToStringMap(next), path[1:])
		case map[string]interface{}:
			// Type assertion is safe here since it is only reached
			// if the type of `next` is the same as the type being asserted
			return searchMap(x, path[1:])
		case []interface{}:
			if !isSlice {
				return nil
			}
			var (
				l = len(idxs) - 1
				m map[string]interface{}
			)
			for i, idx := range idxs {
				if i < l {
					if idx < len(x) {
						nx, ok := x[idx].([]interface{})
						if !ok {
							return nil
						}
						x = nx
					} else {
						return nil
					}
				} else {
					if idx < len(x) {
						m, ok = x[idx].(map[string]interface{})
						if !ok {
							if len(path) > 1 {
								return nil
							} else {
								return x[idx]
							}
						}
					} else {
						return nil
					}
				}
			}
			return searchMap(m, path[1:])
		default:
			// got a value but nested key expected, return "nil" for not found
			return nil
		}
	}
	return nil
}

func isNum(s string) (int, bool) {
	i, err := strconv.Atoi(s)
	if err != nil {
		return 0, false
	}
	return i, true
}

var regIdxes = regexp.MustCompile(`(\w+)((\[\w+\])*)+`)

func isIndex(s string) (k string, idx []int, ok bool) {
	result := regIdxes.FindAllStringSubmatch(s, -1)
	if len(result) == 0 {
		return s, nil, false
	}
	row := result[0]
	if row[3] == "" {
		return s, nil, false
	}
	ok = len(row) == 4
	k = row[1]

	f := func(c rune) bool {
		return c == '[' || c == ']'
	}

	ss := strings.FieldsFunc(row[2], f)

	idx = make([]int, len(ss))
	for j, s := range ss {
		i, err := strconv.Atoi(s)
		if err != nil {
			continue
		}
		idx[j] = i
	}
	return
}

func makeSlice(multi int, len, cap int) interface{} {
	switch multi {
	case 0:
		return nil
	case 1:
		return make([]interface{}, len, cap)
	case 2:
		return make([][]interface{}, len, cap)
	case 3:
		return make([][][]interface{}, len, cap)
	case 4:
		return make([][][][]interface{}, len, cap)
	case 5:
		return make([][][][][]interface{}, len, cap)
	default:
		panic("out of 5 dim")
	}
}

func deepSearch(v interface{}, p interface{}, pk interface{}, paths []string) interface{} {
	if len(paths) == 0 {
		return v
	}

	var k = paths[0]

	paths = paths[1:]
	switch x := v.(type) {
	case map[string]interface{}:
		ak, idxs, ok := isIndex(k)
		if !ok {
			m, ok := x[k]
			if !ok {
				m = make(map[string]interface{})
				x[k] = m
			}
			return deepSearch(m, x, k, paths)
		} else {
			a, ok := x[ak].([]interface{})
			if !ok {
				var (
					ll     = len(idxs) - 1
					i, idx int
					pidx   int
					p      interface{}
				)
				a = make([]interface{}, idx+1)
				x[ak] = a
				// m = makeSlice(len(idxs), l, l)

				// x[ak] = m
				for i, idx = range idxs {
					ca := make([]interface{}, idx+1)
					if i < ll {
						if i == 0 {
							x[ak] = ca
						} else {
							a[pidx] = ca
						}
						pidx = idx
						a = ca
					} else {
						if i == 0 {
							x[ak] = ca
							p = x
						} else {
							a[pidx] = ca
							p = a
						}
						a = ca
					}
				}
				return deepSearch(a, p, idx, paths)
			} else {

				var (
					i, idx int
					ll     = len(idxs) - 1
					pa     = a
					pidx   int
				)

				for i, idx = range idxs {
					if i < ll {
						if idx < len(a) {
							ca, ok := a[idx].([]interface{})
							if !ok {
								ca = make([]interface{}, idx+1)
								a[idx] = ca
							}
							pidx = idx
							pa = a
							a = ca

							// m = make(map[string]interface{})
							// a[idx[0]] = m
							// x[ak] = a
							// return deepSearch(m, x, idx, paths)
						} else {
							l := idx + 1
							ca := make([]interface{}, l)
							copy(ca, a)
							if i == 0 {
								x[ak] = ca
							} else {
								a[idx] = ca
							}
							pidx = idx
							pa = a
							a = ca
							// m = makeSlice(len(idxs), l, l)
							// copy(m.([]interface{}), a)
							// x[ak] = m
							// return deepSearch(m, x, idx, paths)
						}
					} else {
						if idx >= len(a) {
							l := idx + 1
							ca := make([]interface{}, l)
							copy(ca, a)
							if i == 0 {
								x[ak] = ca
							} else {
								pa[pidx] = ca
								// a[idx] = ca
							}
							pa = a
							a = ca
						}
					}
				}

				return deepSearch(a, pa, idx, paths)

			}
		}
	case []interface{}:
		switch pkx := pk.(type) {
		case string:
			mx := make(map[string]interface{})
			if mv, ok := p.(map[string]interface{}); ok {
				mv[pkx] = mx
				if m, ok := mx[k]; ok {
					return deepSearch(m, mx, k, paths)
				} else {
					m = make(map[string]interface{})
					mx[k] = m
					return deepSearch(m, mx, k, paths)
				}
			}
		case int:
			v := x[pkx]
			switch mx := v.(type) {
			case map[string]interface{}:
				if m, ok := mx[k]; ok {
					return deepSearch(m, mx, k, paths)
				} else {
					m = make(map[string]interface{})
					mx[k] = m
					return deepSearch(m, mx, k, paths)
				}
			default:
				var mv = make(map[string]interface{})
				x[pkx] = mv
				m := make(map[string]interface{})
				mv[k] = m
				return deepSearch(m, mv, k, paths)
			}
		}
		// default:
		// 	switch pp := p.(type) {
		// 	case map[string]interface{}:
		// 		if pkk, ok := pk.(string); ok {
		// 			x := make(map[string]interface{})
		// 			pp[pkk] = x
		// 			m := make(map[string]interface{})
		// 			x[k] = m
		// 			deepSearch(m, x, k, paths)
		// 		}
		// 	case []interface{}:
		// 		if pkk, ok := pk.(int); ok {
		// 			if i, ok := isNum(k); ok {
		// 				x := make([]interface{}, 0)
		// 				pp[pkk] = x
		// 				m := make(map[string]interface{})
		// 				if i < len(x) {
		// 					x[i] = m
		// 				} else {
		// 					x = make([]interface{}, i+1)
		// 					x[i] = m
		// 				}
		// 				deepSearch(m, x, i, paths)
		// 			} else {
		// 				x := make(map[string]interface{})
		// 				pp[pkk] = x
		// 				m := make(map[string]interface{})
		// 				x[k] = m
		// 				deepSearch(m, x, k, paths)
		// 			}
		// 		}
		// 	default:
		// 		return p
		// 	}
	}
	return v
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
			m = reflect.Indirect(m)

			switch m.Kind() {
			case reflect.Map:
				pkey = reflect.ValueOf(s)
				mv := m.MapIndex(pkey)
				if mv.IsValid() {
					return elemval(mv), true
				}
				return mv, false
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
		m = reflect.Indirect(m)

		switch m.Kind() {
		case reflect.Map:
			pkey = reflect.ValueOf(s)
			mm := m.MapIndex(pkey)
			if mm.IsValid() {
				m = elemval(mm)
			} else {
				return mm, false
			}
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

func Set(v interface{}, selector string, val interface{}, opts ...SetOptFunc) error {
	var (
		m, ok = v.(map[string]interface{})
		paths = strings.Split(selector, ".")
	)
	if !ok {
		return errors.New("invalid map type")
	}
	// mv := deepSearch(m, paths[:len(paths)-1])
	// lastKey := strings.ToLower(paths[len(paths)-1])
	lastKey := paths[len(paths)-1]
	_, idx, lastSlice := isIndex(lastKey)
	if !lastSlice {
		paths = paths[:len(paths)-1]
	}
	mv := deepSearch(m, nil, nil, paths)

	switch x := mv.(type) {
	case map[string]interface{}:
		x[lastKey] = val
	case []interface{}:
		if lastSlice {
			x[idx[0]] = val
		} else {
			_, idx, _ = isIndex(paths[len(paths)-1])
			mv, ok := x[idx[0]].(map[string]interface{})
			if !ok {
				mv = make(map[string]interface{})
				x[idx[0]] = mv
			}
			mv[lastKey] = val
		}
	}
	// mv[lastKey] = val

	return nil
}

func Get(m interface{}, selector string) interface{} {
	// v := reflect.ValueOf(m)
	paths := strings.Split(selector, ".")
	switch x := m.(type) {
	case map[string]interface{}:
		return searchMap(x, paths)
	case *map[string]interface{}:
		return searchMap(*x, paths)
	default:
		return nil
	}
	// if val, ok := getIndexPath(v, selector); ok {
	// 	if val.IsValid() {
	// 		return val.Interface()
	// 	} else {
	// 		return nil
	// 	}
	// }
	// return nil
}
