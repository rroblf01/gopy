from email.mime.multipart import MIMEMultipart
from email.mime.text import MIMEText


def main() -> None:
    root = MIMEMultipart("mixed", "BOUND")
    root.attach(MIMEText("first", "plain", "us-ascii"))
    root.attach(MIMEText("second", "plain", "us-ascii"))
    print(root.get("Content-Type"))
    print(root.is_multipart())
    print(root.get_content_maintype())
    print(root.get_content_subtype())


if __name__ == "__main__":
    main()
