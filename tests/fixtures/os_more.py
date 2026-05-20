import os
import os.path


def main() -> None:
    pre = os.path.split("/tmp/sub/file.txt")
    for x in pre:
        print(x)
    print(os.path.split("file.txt")[0])
    print(os.path.split("file.txt")[1])
    print(os.path.relpath("/a/b/c", "/a"))
    print(os.path.relpath("a/b/c", "a"))


if __name__ == "__main__":
    main()
