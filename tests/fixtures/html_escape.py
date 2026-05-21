import html


def main() -> None:
    print(html.escape("<a href=\"x\">Tom & Jerry</a>"))
    print(html.escape("'quoted'"))
    print(html.unescape("&lt;b&gt;bold&lt;/b&gt;"))
    print(html.unescape("Tom &amp; Jerry"))


if __name__ == "__main__":
    main()
