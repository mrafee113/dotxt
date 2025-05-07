package config

const DefaultConfig = `
colors:
  black: &black "#000000"
  red: &red "#B61C1C"
  light-orange: &light-orange "#e88761"
  green: &green "#21FC00"
  blue: &blue "#4895bf"
  purple: &purple "#9e64ea"
  cyan: &cyan "#05FCC6"
  light-jade: &light-jade "#6aa88f"
  light-grey: &light-grey "#919191"
  dark-grey: &dark-grey "#4C4C4C"
  light-red: &light-red "#FC7876"
  light-green: &light-green "#78FC76"
  yellow: &yellow "#FCFC64"
  light-yellow: &light-yellow "#f4f07f"
  pale-yellow: &pale-yellow "#f9f7b9"
  dark-yellow: &dark-yellow "#b2bc45"
  light-blue: &light-blue "#6CC0FC"
  light-purple: &light-purple "#d994fc"
  light-cyan: &light-cyan "#9FFCF3"
  light-pink: &light-pink "#fc8ae1"
  brown: &brown "#c48660"
  white: &white "#FFFFFF"
  dark-white: &dark-white "#c4c4c4"
  default: &default "#DEF4ED"

print:
  color-header: *light-red
  color-default: *default
  color-index: *light-grey

  color-date-due: *light-red
  color-date-end: *light-red
  color-date-dead: *light-red   # deadline
  color-date-r: *light-jade     # reminders
  color-every: *light-yellow

  color-at: *blue
  color-plus: *light-jade
  color-tag: *light-pink

  ids:
    saturation: 0.7
    lightness: 0.6
    start-hue: 30
    end-hue: 210

  progress:
    count: *default
    done-count: *light-grey
    percentage:
      saturation: 0.7
      lightness: 0.6
    unit: *default
    bartext-len: 10
    header: *pale-yellow

  priority:
    saturation: 0.7
    lightness: 0.6
    group-depth: 5
    start-hue: 0
    end-hue: 360
  
  temporal-format:
    c: rn
    lud: rn
    due: rn
    end: due
    dead: due
    r: rn
`
