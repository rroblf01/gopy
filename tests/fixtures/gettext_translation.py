import gettext


def main() -> None:
    t = gettext.translation("myapp", fallback=True)
    print(t.gettext("hello"))
    print(t.ngettext("apple", "apples", 1))
    print(t.ngettext("apple", "apples", 5))


if __name__ == "__main__":
    main()
