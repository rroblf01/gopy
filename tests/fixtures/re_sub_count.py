import re
import warnings


def main() -> None:
    warnings.filterwarnings("ignore", category=DeprecationWarning)
    print(re.sub("a", "X", "banana"))
    print(re.sub("a", "X", "banana", 1))
    print(re.sub("a", "X", "banana", 2))
    print(re.sub("a", "X", "banana", 0))


if __name__ == "__main__":
    main()
