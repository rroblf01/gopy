import os
import os.path


def main() -> None:
    print(os.path.normpath("/a/b/../c/./d"))
    print(os.path.normpath("a//b/c/"))
    e = os.path.expanduser("~/foo")
    print(len(e) > 0)
    print(os.path.expandvars("$HOME/x") != "$HOME/x")
    print(os.path.commonprefix(["/a/b/c", "/a/b/d", "/a/b/e"]))
    print(os.path.samefile("/tmp", "/tmp"))


if __name__ == "__main__":
    main()
