import shapes
from shapes import PI


def main() -> None:
    print(round(shapes.circle_area(2.0), 4))
    print(shapes.rect_area(3.0, 4.0))
    print(PI)


if __name__ == "__main__":
    main()
