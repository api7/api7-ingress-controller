# API7 Ingress Controller CRD API参考文档

本目录包含API7 Ingress Controller的CRD API参考文档，由[elastic/crd-ref-docs](https://github.com/elastic/crd-ref-docs)工具生成，使用自定义模板。

## 生成文档

您可以使用以下命令生成CRD参考文档：

### 单文件模式（默认）

生成单个Markdown文件包含所有API参考：

```bash
make generate-crd-docs
```

生成的文档将保存在 `docs/crd/api.md` 文件中。

### 分组模式

按API组分别生成多个文档文件：

```bash
make generate-crd-docs-grouped
```

生成的文档将保存在 `docs/crd/groups/` 目录下，每个API组一个文件。

## 配置

文档生成配置位于 `docs/crd/config.yaml` 文件中，您可以根据需要修改此配置文件。

### 自定义模板

文档生成使用 `docs/template` 目录中的自定义模板，这些模板定义了文档的格式和样式：

- `gv_list.tpl` - 主文档模板，包含文档标题和API组列表
- `gv_details.tpl` - API组详细信息模板
- `type.tpl` - 类型定义模板
- `type_members.tpl` - 类型成员模板

若要修改文档的格式或内容，可以编辑这些模板文件。 