def main() -> None:
    a: dict[str, int] = {"x": 1, "y": 2}
    b: dict[str, int] = {"y": 99, "z": 3}
    c = a | b
    print(c["x"])
    print(c["y"])
    print(c["z"])
    print(len(a))  # unchanged
    print("[%d]" % 42)
    print("%s = %d" % ("foo", 7))
    print("pi=%.2f" % 3.14159)
    print(ascii("hi"))
    print(ascii("héllo"))


if __name__ == "__main__":
    main()
