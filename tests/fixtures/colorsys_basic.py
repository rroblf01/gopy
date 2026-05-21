import colorsys


def main() -> None:
    hsv = colorsys.rgb_to_hsv(1.0, 0.0, 0.0)
    print(hsv[2])
    rgb = colorsys.hsv_to_rgb(0.0, 1.0, 1.0)
    print(rgb[0])
    print(rgb[1])
    print(rgb[2])


if __name__ == "__main__":
    main()
