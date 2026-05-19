import { Modal } from "../common/Modal";
import { OpenExternalURL } from "../../../wailsjs/go/backend/Service";

export function FAQModal({ open, onClose }: { open: boolean; onClose: () => void }) {
  return (
    <Modal open={open} onClose={onClose} title="常见问题" width={520}>
      <div className="faq">
        <details open>
          <summary>API Key 应该选哪个分组?</summary>
          <p>
            本应用调用的是上游的 <code>/v1/responses</code> 接口(类似 OpenAI Responses API),
            而不是 <code>/v1/images/generations</code>。图像生成是通过模型内置的
            <code> image_generation </code> 工具触发的。
          </p>
          <p>
            所以在 GPTCODEX 中转站后台,你的 key 需要绑定到
            <strong>「拥有 gpt-5.5 模型的分组」</strong>:
          </p>
          <ul>
            <li>✅ 推荐选 <strong>「余额分组」</strong> 或 <strong>「套餐分组」</strong></li>
            <li>❌ <strong>不要</strong>选「image-2 分组」 — 那是直接 image API 的分组,不包含 gpt-5.5</li>
          </ul>
          <p>
            如果 key 没有 gpt-5.5 权限,接口会返回 401/403 或者 "model not found" 错误。
          </p>
        </details>

        <details>
          <summary>支持哪些上游中转站?</summary>
          <p>
            默认 <code>https://gptcodex.top</code>。任何兼容 OpenAI <strong>Responses API</strong> 形态 + 提供
            <code> image_generation </code> 工具的中转站理论上都行。在「设置 → BASE_URL」可以切换。
          </p>
          <p>
            注意:OpenAI 官方 <code>/v1/chat/completions</code> 接口的中转站
            <strong>不兼容</strong>(本应用不发 chat completions 请求)。
          </p>
        </details>

        <details>
          <summary>能换其他文本 / 图像模型吗?</summary>
          <p>
            可以。在「设置」展开「文本模型 ID」和「图像模型 ID」填上即可。
          </p>
          <ul>
            <li><strong>文本模型 ID</strong>:默认 <code>gpt-5.5</code>(用于推理 + 调用 image 工具)</li>
            <li><strong>图像模型 ID</strong>:默认 <code>gpt-image-2</code>(实际生成图片的工具)</li>
          </ul>
        </details>

        <details>
          <summary>生成失败 / 504 / 524 怎么办?</summary>
          <p>
            上游网关超时(Cloudflare 504/524)在中转站上很常见。本应用<strong>自动重试 3 次,每次间隔 15 秒</strong>。
            如果三次都失败:
          </p>
          <ul>
            <li>检查 key 是否过期 / 余额是否充足 / 是否绑对了分组(见第一条)</li>
            <li>试试切换网络通道:「设置 → 网络通道」改成 <code>curl</code> 让请求走系统 curl,有时能绕过原生 HTTP 的 TLS 问题</li>
            <li>查看历史项右键「📄 查看 raw 响应」看上游具体返回了什么</li>
          </ul>
        </details>

        <details>
          <summary>蒙版 / 多参考图 / seed 上游会用吗?</summary>
          <p>
            这些字段在请求 payload 里都会发送,但<strong>上游是否真的使用取决于中转站和模型的实现</strong>:
          </p>
          <ul>
            <li><strong>多参考图</strong>:作为多个 <code>input_image</code> 内容块发送,上游解释方式因模型而异</li>
            <li><strong>蒙版</strong>:作为 tool 的 <code>mask</code> 字段发送,gpt-image-2 通常会用但不保证 100%</li>
            <li><strong>seed</strong>:作为 tool 的 <code>seed</code> 字段发送,理论上能复现结果</li>
            <li><strong>negative prompt</strong>:作为 tool 的 <code>negative_prompt</code> 字段发送,部分实现支持</li>
          </ul>
        </details>

        <details>
          <summary>数据存在哪里?会上传吗?</summary>
          <p>
            <strong>完全本地存储,不上传任何服务器</strong>(除了向上游 API 转发你的生成请求):
          </p>
          <ul>
            <li>API Key:<code>localStorage</code></li>
            <li>历史记录元数据:浏览器 IndexedDB</li>
            <li>生成的图片 PNG:<code>%APPDATA%\image-studio\images\</code></li>
            <li>导入的源图:<code>%APPDATA%\image-studio\imports\</code></li>
            <li>原始 SSE 响应:跟 PNG 同目录,排错时用</li>
          </ul>
        </details>

        <details>
          <summary>快捷键?</summary>
          <ul>
            <li><kbd>Ctrl</kbd>+<kbd>Enter</kbd> — 提交生成</li>
            <li><kbd>Ctrl</kbd>+<kbd>N</kbd> / <kbd>Ctrl</kbd>+<kbd>W</kbd> — 新建 / 关闭标签</li>
            <li><kbd>Ctrl</kbd>+<kbd>Z</kbd> / <kbd>Ctrl</kbd>+<kbd>Shift</kbd>+<kbd>Z</kbd> — 撤销 / 重做</li>
            <li><kbd>Ctrl</kbd>+<kbd>C</kbd> / <kbd>Ctrl</kbd>+<kbd>V</kbd> — 复制 / 粘贴图片</li>
            <li><kbd>1</kbd> / <kbd>2</kbd> / <kbd>3</kbd> — 拖动 / 蒙版 / 标注 工具</li>
            <li><kbd>空格</kbd> — 按住临时切到拖动</li>
            <li><kbd>F</kbd> — 重置视图;双击画板 — fit ↔ 100%</li>
            <li><kbd>F11</kbd> — 全屏</li>
            <li><kbd>[</kbd> / <kbd>]</kbd> — 笔刷大小</li>
            <li><kbd>Esc</kbd> — 取消生成 / 退出对比 / 关闭弹窗</li>
            <li><kbd>Delete</kbd> — 删除选中标注</li>
          </ul>
        </details>

        <details>
          <summary>反馈渠道?</summary>
          <p>
            <a
              style={{ color: "var(--accent)", cursor: "pointer", textDecoration: "underline" }}
              onClick={() => OpenExternalURL("https://github.com/RoseKhlifa/Image-Studio/issues").catch(() => undefined)}
            >GitHub Issues</a> · 项目 MIT 协议开源
          </p>
        </details>
      </div>
    </Modal>
  );
}
