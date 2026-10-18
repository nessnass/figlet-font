# Nessnass figlet font

A [figlet](http://www.figlet.org/) font drawn with single-line Unicode
box-drawing characters. Three rows tall, for terminal banners and headers.
By [Nessnass](https://www.nessnass.com/?utm_source=github.com).

```
┌┐╷┌─╴┌─┐┌─┐┌┐╷┌─┐┌─┐┌─┐
│││├─ └─┐└─┐│││├─┤└─┐└─┐
╵└┘└─╴└─┘└─┘╵└┘╵ ╵└─┘└─┘
```

## Use

```sh
figlet -d . -f Nessnass "Hello"
```

Or copy `Nessnass.flf` into your figlet font directory.

## Characters

```
┌─┐┌─╮┌─┐┌─╮┌─╴┌─╴┌─┐╷ ╷╷  ╷╷┌─╷  ┌┬┐
├─┤├─┤│  │ │├─ ├─ │ ┐├─┤│  │├┴┐│  │││
╵ ╵└─╯└─┘└─╯└─╴╵  └─┘╵ ╵╵└─┘╵ ╵└─╴╵ ╵
┌┐╷┌─┐┌─┐┌─┐┌─╮┌─┐─┬─╷ ╷╶┐┌╴╷ ╷╶┐┌╷ ╷╶─┐
││││ │├─┘│ │├┬╯└─┐ │ │ │ ││ │││┌┼┘└┬┘┌─┘
╵└┘└─┘╵  └─┼╵└╴└─┘ ╵ └─┘ └┘ └┴┘┘└╴ ╵ └─╴
╭─╮╶┐╭─╮┌─┐╷ ╷┌─╴┌─┐┌─┐╭─╮┌─┐
│╱│ │╭─┘ ─┤└─┤└─┐├─┐  │├─┤└─┤
╰─╯ ╵└─╴└─┘  ╵└─┘└─┘  ╵╰─╯  ╵
╷  ╷╷  ─┼─┼─  ┌┼┐  O╷   ┌   ╷  ╭  ╮  \│/   ╷
│      ─┼─┼─  └┼┐  ┌┘  ┌┼─     │  │  ─ ─  ─┼─
·             └┼┘  ╵O  └┘      ╰  ╯  /│\   ╵
             ╱        ╱       ╲  ╭─╮  ╭─╮
   ───          ·  ·     ───      ╭╯  │╰╯
┘       ·  ╱    ·  ┘  ╲  ───  ╱   ·   ╰──
┌─  ╲    ─┐  /\        ┐  ╭  │  ╮  ╭╮
│         │               ┤  │  ├   ╰╯
└─    ╲  ─┘      ────     ╰  │  ╯
```

Upper and lower case are drawn the same. The zero is slashed so it is never
read as an O. An en space (U+2002) is a half space, for fine positioning.

## Viewer

A small Go app in `viewer/` that shows the font in the terminal as you type.

```sh
cd viewer
go build
./nessnass
```

## Credits

Based on the Calvin S figlet font.

## Licence

[MIT](LICENSE)
