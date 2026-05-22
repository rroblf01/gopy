import fnmatch
import re


def main() -> None:
    pat = fnmatch.translate("*.py")
    rx = re.compile(pat)
    print(rx.match("foo.py") is not None)
    print(rx.match("foo.txt") is not None)


if __name__ == "__main__":
    main()
