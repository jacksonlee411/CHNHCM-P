import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { createBrowserRouter, RouterProvider } from "react-router-dom";
import { App } from "./app/App";

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <RouterProvider
      router={createBrowserRouter([{ path: "*", element: <App />, errorElement: <main role="alert"><h1>页面暂时无法显示</h1><p>请返回首页后重试。</p><a href="/">返回首页</a></main> }])}
    />
  </StrictMode>,
);
