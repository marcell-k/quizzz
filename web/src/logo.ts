// Logo from ui/logo.png next to the exe. Missing file removes the image,
// so pages keep their text title.
export function logo(className = ""): HTMLImageElement {
  const img = document.createElement("img");
  img.className = className === "" ? "logo" : `logo ${className}`;
  img.src = "/media/ui/logo.png";
  img.alt = "";
  img.style.visibility = "hidden";
  img.onload = () => {
    img.style.visibility = "visible";
  };
  img.onerror = () => img.remove();
  return img;
}
