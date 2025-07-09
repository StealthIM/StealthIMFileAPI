package errorcode

const (
	// FSAPInternalError 一般内部错误
	FSAPInternalError int32 = 1100 + iota
	// FSAPFileNotFound 文件不存在
	FSAPFileNotFound
	// FSAPMetadataEmpty 元数据为空
	FSAPMetadataEmpty
	// FSAPMetadataError 元数据错误
	FSAPMetadataError
	// FSAPHashBroken 文件哈希值不匹配
	FSAPHashBroken
	// FSAPFileEmpty 文件为空
	FSAPFileEmpty
	// FSAPSaveBlockStorageFault 保存块存储错误
	FSAPSaveBlockStorageFault
	// FSAPHashNotMatch 文件哈希值不匹配
	FSAPHashNotMatch
	// FSAPUploadToDatabaseFault 上传到数据库错误
	FSAPUploadToDatabaseFault
	// FSAPUploadBlockRepeat 上传文件块重复
	FSAPUploadBlockRepeat
)
