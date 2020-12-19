# 直接存取json

用反射封装解析json的函数

## 带回源函数的批量查询函数`BizMGetJsonMapWithFill`
很常见的场景是批量缓存，缓存可能一部分发生修改（常见做法是修改后执行del key），就无法命中缓存,
未命中的部分，需要回源，然后将回源结果写入缓存，最后将命中和未命中的两部分合并返回给用户。

### 函数入参解释
入参解释：
1. `bizKeys`必须是`slice`
2. `keyFunc`必须只有一个入参和一个出参，
    * 入参类型必须与`bizKeys`的数组内每个元素的类型相同
    * 出参类型必须是`string`
3. `fillFunc`必须是2个入参，2个出参
    * 入参顺序：
        1. `ctx context.Context`
        2. `noCachedBizKeys` 未命中的`bizKey`数组，类型必须与`bizKeys`相同
    * 出参顺序：
        1. `noCachedMap map[bizKey类型]bizValue类型`
        2. `err error` 发生错误时，会返回
4. `outPtr` 必须是`map[bizKey类型]bizValue类型`的地址
### 函数执行流程
1. 执行`redis mget`获得命中缓存的`cachedMap`和未命中缓存的`noCachedBizKeys`，
2. 用`noCachedBizKeys`调用回源函数`fillFunc`，得到回源结果`noCachedMap`，
3. 将`noCachedMap`写入缓存，并且与`cachedMap`合并给到`outPtr`

### 如何避免击穿
`fillFunc`返回的`noCachedMap`中，将回源也查不到的`bizKey`的`bizValue`填为`nil`
那么会被保存到redis中（`json.Unmarshal`为`null`），下次查redis查到`null`就不会回源了
