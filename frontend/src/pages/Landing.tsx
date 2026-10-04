import { useEffect } from "react";

/** The public homepage is the commerce page in public/market. */
export function Landing(_props: { signedIn?: boolean }) {
  useEffect(() => {
    const previous = document.title;
    document.title = "FinGuard — Financial investigation, with context";
    document.body.classList.add("market-home");
    return () => {
      document.title = previous;
      document.body.classList.remove("market-home");
    };
  }, []);

  return (
    <iframe
      className="market-frame"
      src="/market/index.html?brand=mark"
      title="FinGuard"
    />
  );
}
