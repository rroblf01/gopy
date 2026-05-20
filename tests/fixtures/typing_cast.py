from typing import cast


def main() -> None:
    v = 42
    s = cast(int, v)
    print(s)
    print(cast(str, "hi"))


if __name__ == "__main__":
    main()
