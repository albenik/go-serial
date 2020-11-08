package enumerator

//sys setupDiClassGuidsFromNameInternal(class string, guid *guid, guidSize uint32, requiredSize *uint32) (err error) = setupapi.SetupDiClassGuidsFromNameW
//sys setupDiGetClassDevs(guid *guid, enumerator *string, hwndParent uintptr, flags uint32) (set hDevInfo, err error) = setupapi.SetupDiGetClassDevsW
//sys setupDiDestroyDeviceInfoList(set hDevInfo) (err error) = setupapi.SetupDiDestroyDeviceInfoList
//sys setupDiEnumDeviceInfo(set hDevInfo, index uint32, info *pspDevInfoData) (err error) = setupapi.SetupDiEnumDeviceInfo
//sys setupDiGetDeviceInstanceId(set hDevInfo, devInfo *pspDevInfoData, devInstanceId unsafe.Pointer, devInstanceIdSize uint32, requiredSize *uint32) (err error) = setupapi.SetupDiGetDeviceInstanceIdW
//sys setupDiOpenDevRegKey(set hDevInfo, devInfo *pspDevInfoData, scope dicsScope, hwProfile uint32, keyType uint32, samDesired regsam) (hkey syscall.Handle, err error) = setupapi.SetupDiOpenDevRegKey
//sys setupDiGetDeviceRegistryProperty(set hDevInfo, devInfo *pspDevInfoData, property deviceProperty, propertyType *uint32, outValue *byte, outSize uint32, reqSize *uint32) (res bool) = setupapi.SetupDiGetDeviceRegistryPropertyW
