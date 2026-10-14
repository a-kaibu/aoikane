package phrase

// Part-of-speech predicates of the native engine. The names carry the address
// of the original function so the port can be checked against the binary.

// p16b750: verb stem — class 3 with the secondary class 0x600.
func (m *Morph) p16b750() bool {
	return m.POS&0xff000000 == 0x3000000 && m.POS2 == 0x600
}

// p16b780: class 8 or 9, or the specific code 0x3020000.
func (m *Morph) p16b780() bool {
	v := m.POS&0xff000000 - 0x8000000
	if v&0xfeffffff == 0 {
		return true
	}
	return m.POS == 0x3020000
}

// p16b7b0: suffix-like nouns and the specific codes of the original.
func (m *Morph) p16b7b0() bool {
	p := m.POS
	return p&0xffff0000 == 0x1070000 || p == 0x1080000 || p == 0x10b0000 || p == 0x3010000 || p == 0x4010000
}

// p16b7f0: content word.
func (m *Morph) p16b7f0() bool {
	p := m.POS
	if p == 0xd030000 || p&0xffff0000 == 0x1070000 || p == 0x1080000 || p == 0x10b0000 || p == 0x3010000 || p == 0x4010000 {
		return false
	}
	class := p & 0xff000000
	if class == 0x1000000 {
		if p&0xffff0000 != 0x1090000 {
			return true
		}
	} else {
		if class == 0x3000000 && !m.p16b980() {
			return true
		}
		if class == 0x4000000 && !m.p16b980() {
			return true
		}
		if (class+0xfb000000)&0xfeffffff == 0 {
			return true
		}
	}
	if p != 0x7000000 && p != 0xa000000 && p != 0xb000000 && p != 0xc000000 {
		return class == 0xd000000
	}
	return true
}

// p16b950: noun class that is not the 0x1090000 group.
func (m *Morph) p16b950() bool {
	return m.POS&0xff000000 == 0x1000000 && m.POS&0xffff0000 != 0x1090000
}

// p16b980: the 0x1090000 group or the specific codes 0x3020000 / 0x4020000.
func (m *Morph) p16b980() bool {
	p := m.POS
	return p&0xffff0000 == 0x1090000 || p == 0x3020000 || p == 0x4020000
}

// p16b900 inspects the tertiary class: 0x300 or 0x900.
func (m *Morph) p16b900() bool {
	v := m.POS3 & 0xf00
	return v == 0x300 || v == 0x900
}

// p16b920 inspects the tertiary class: 0xa00, 0xa01, 0xc00 or 0xc01.
func (m *Morph) p16b920() bool {
	v := m.POS3 - 0xa00
	return v&0xfffffdfe == 0 || m.POS3 == 0xc01
}

// p16e680: no reading and no tag text.
func (m *Morph) p16e680() bool { return m.silent() }
