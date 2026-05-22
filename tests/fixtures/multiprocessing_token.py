import multiprocessing
import token


def main() -> None:
    print(multiprocessing.cpu_count() > 0)
    print(token.NAME)
    print(token.NUMBER)
    print(token.STRING)
    print(token.OP)


if __name__ == "__main__":
    main()
