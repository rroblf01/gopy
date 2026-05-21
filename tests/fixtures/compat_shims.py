import weakref
import traceback


class Box:
    def __init__(self, n: int) -> None:
        self.n = n


def main() -> None:
    b = Box(7)
    r = weakref.ref(b)
    again = r()
    if isinstance(again, Box):
        print(again.n)
    msg: str = traceback.format_exc()
    print(len(msg) > 0)


if __name__ == "__main__":
    main()
