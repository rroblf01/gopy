import textwrap


def main() -> None:
    lines = textwrap.wrap("the quick brown fox jumps over the lazy dog", 15)
    print(len(lines))
    for line in lines:
        print(line)
    print(textwrap.shorten("hello world this is a long sentence", 20))


if __name__ == "__main__":
    main()
