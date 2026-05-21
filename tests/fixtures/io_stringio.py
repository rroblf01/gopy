import io


def main() -> None:
    s = io.StringIO()
    s.write("hello ")
    s.write("world")
    print(s.getvalue())
    print(s.tell())

    s2 = io.StringIO("initial ")
    s2.seek(s2.tell())
    s2.write("data")
    print(s2.getvalue())

    s.close()


if __name__ == "__main__":
    main()
