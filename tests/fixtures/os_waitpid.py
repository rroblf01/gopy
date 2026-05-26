import os


def main() -> None:
    print(os.WIFEXITED(0))
    print(os.WEXITSTATUS(256 * 7))
    print(os.WIFSIGNALED(0))
    print(os.WTERMSIG(0))


if __name__ == "__main__":
    main()
