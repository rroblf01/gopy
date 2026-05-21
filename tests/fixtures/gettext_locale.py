import gettext
import locale


def main() -> None:
    print(gettext.gettext("hello"))
    print(gettext.ngettext("apple", "apples", 1))
    print(gettext.ngettext("apple", "apples", 5))
    loc = locale.getlocale()
    print(len(loc) >= 2)
    print(locale.setlocale(locale.LC_ALL, "C"))


if __name__ == "__main__":
    main()
