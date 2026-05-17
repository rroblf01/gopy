@staticmethod
def double(x: int) -> int:
    return x * 2


def main() -> None:
    # @staticmethod on a free function is a no-op in both Python and gopy.
    # We accept and ignore the decorator so transpiling annotated code
    # doesn't fail spuriously.
    print(double(21))


if __name__ == "__main__":
    main()
