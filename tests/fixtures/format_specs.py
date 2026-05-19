def main() -> None:
    print("[{}]".format(42))
    print("[{:5d}]".format(42))
    print("[{:05d}]".format(42))
    print("[{:.2f}]".format(3.14159))
    print("[{:8.2f}]".format(3.14159))
    print("[{:>10}]".format("hi"))
    print("[{:<10}]".format("hi"))
    print("[{:^10}]".format("hi"))
    print("[{:*>6}]".format("ab"))
    print("[{:-<6}]".format("ab"))
    print("[{:x}]".format(255))
    print("[{:08x}]".format(255))
    print("[{:b}]".format(5))
    print("({}, {})".format("a", "b"))


if __name__ == "__main__":
    main()
