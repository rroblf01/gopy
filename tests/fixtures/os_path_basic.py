import os
import os.path


def main() -> None:
    p = os.path.join("a", "b", "c.txt")
    print(p)
    print(os.path.basename("/tmp/file.txt"))
    print(os.path.dirname("/tmp/sub/file.txt"))
    parts = os.path.splitext("/tmp/file.txt")
    for x in parts:
        print(x)
    print(os.path.exists("/tmp"))
    print(os.path.isdir("/tmp"))
    print(os.path.isfile("/tmp"))
    print(os.path.exists("/nonexistent_path_gopy"))


if __name__ == "__main__":
    main()
