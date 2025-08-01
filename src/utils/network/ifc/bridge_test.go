package ifc

import (
	"errors"
	"net"
	"syscall"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// LinkManagerMock uses testify's mock.Mock
type LinkManagerMock struct {
	mock.Mock
}

func (m *LinkManagerMock) Exists(name string) (bool, error) {
	args := m.Called(name)
	return args.Bool(0), args.Error(1)
}
func (m *LinkManagerMock) SetMaster(name string, masterName string) error {
	return m.Called(name, masterName).Error(0)
}
func (m *LinkManagerMock) AddLink(name string, typ LinkType) error {
	return m.Called(name, typ).Error(0)
}
func (m *LinkManagerMock) SetIP(name string, ip net.IP, mask net.IPMask) error {
	return m.Called(name, ip, mask).Error(0)
}
func (m *LinkManagerMock) BringUp(name string) error {
	return m.Called(name).Error(0)
}
func (m *LinkManagerMock) DeleteLink(name string) error {
	return m.Called(name).Error(0)
}
func (m *LinkManagerMock) HasIP(name string, ip net.IP, mask net.IPMask) (bool, error) {
	args := m.Called(name, ip, mask)
	return args.Bool(0), args.Error(1)
}
func (m *LinkManagerMock) DisableTxOffloading(name string) error {
	return m.Called(name).Error(0)
}

func TestCreateBridgeWithManager_Success(t *testing.T) {
	mgr := &LinkManagerMock{}
	mgr.On("AddLink", "br0", LinkTypeBridge).Return(nil)
	mgr.On("SetIP", "br0", mock.Anything, mock.Anything).Return(nil)
	mgr.On("BringUp", "br0").Return(nil)
	mgr.On("DisableTxOffloading", "br0").Return(nil)
	mgr.On("HasIP", "br0", mock.Anything, mock.Anything).Return(true, nil)
	err := CreateBridgeWithManager(mgr, "br0", "192.168.1.0/24", false)
	require.NoError(t, err)
}

func TestCreateBridgeWithManager_AddLinkExistsWithIP(t *testing.T) {
	mgr := &LinkManagerMock{}
	mgr.On("AddLink", "br0", LinkTypeBridge).Return(syscall.EEXIST)
	mgr.On("HasIP", "br0", mock.Anything, mock.Anything).Return(true, nil)
	err := CreateBridgeWithManager(mgr, "br0", "192.168.1.0/24", false)
	require.NoError(t, err)
}

func TestCreateBridgeWithManager_AddLinkExistsWithoutIP(t *testing.T) {
	mgr := &LinkManagerMock{}
	mgr.On("AddLink", "br0", LinkTypeBridge).Return(syscall.EEXIST)
	mgr.On("HasIP", "br0", mock.Anything, mock.Anything).Return(false, nil)
	mgr.On("SetIP", "br0", mock.Anything, mock.Anything).Return(errors.New("set ip failed"))
	mgr.On("DeleteLink", "br0").Return(nil)
	err := CreateBridgeWithManager(mgr, "br0", "192.168.1.0/24", false)
	require.Error(t, err)
}

func TestCreateBridgeWithManager_AddLinkError(t *testing.T) {
	mgr := &LinkManagerMock{}
	mgr.On("AddLink", "br0", LinkTypeBridge).Return(errors.New("add link failed"))
	err := CreateBridgeWithManager(mgr, "br0", "192.168.1.0/24", false)
	require.EqualError(t, err, "failed to add bridge br0: add link failed")
}

func TestCreateBridgeWithManager_SetIPErrorAndDeleteLinkSuccess(t *testing.T) {
	mgr := &LinkManagerMock{}
	mgr.On("AddLink", "br0", LinkTypeBridge).Return(nil)
	mgr.On("SetIP", "br0", mock.Anything, mock.Anything).Return(errors.New("set ip failed"))
	mgr.On("DeleteLink", "br0").Return(nil)
	err := CreateBridgeWithManager(mgr, "br0", "192.168.1.0/24", false)
	require.EqualError(t, err, "failed to set ip: set ip failed")
}

func TestCreateBridgeWithManager_SetIPErrorAndDeleteLinkError(t *testing.T) {
	mgr := &LinkManagerMock{}
	mgr.On("AddLink", "br0", LinkTypeBridge).Return(nil)
	mgr.On("SetIP", "br0", mock.Anything, mock.Anything).Return(errors.New("set ip failed"))
	mgr.On("DeleteLink", "br0").Return(errors.New("delete link failed"))
	err := CreateBridgeWithManager(mgr, "br0", "192.168.1.0/24", false)
	require.EqualError(t, err, "failed to set ip: set ip failed, failed to delete link: delete link failed")
}

func TestCreateBridgeWithManager_BringUpErrorAndDeleteLinkSuccess(t *testing.T) {
	mgr := &LinkManagerMock{}
	mgr.On("AddLink", "br0", LinkTypeBridge).Return(nil)
	mgr.On("SetIP", "br0", mock.Anything, mock.Anything).Return(nil)
	mgr.On("BringUp", "br0").Return(errors.New("bring up failed"))
	mgr.On("DeleteLink", "br0").Return(nil)
	err := CreateBridgeWithManager(mgr, "br0", "192.168.1.0/24", false)
	require.EqualError(t, err, "failed to bring bridge br0 up: bring up failed")
}

func TestCreateBridgeWithManager_BringUpErrorAndDeleteLinkError(t *testing.T) {
	mgr := &LinkManagerMock{}
	mgr.On("AddLink", "br0", LinkTypeBridge).Return(nil)
	mgr.On("SetIP", "br0", mock.Anything, mock.Anything).Return(nil)
	mgr.On("BringUp", "br0").Return(errors.New("bring up failed"))
	mgr.On("DeleteLink", "br0").Return(errors.New("delete link failed"))
	err := CreateBridgeWithManager(mgr, "br0", "192.168.1.0/24", false)
	require.EqualError(t, err, "failed to bring bridge br0 up: bring up failed, failed to delete link: delete link failed")
}

func TestCreateBridgeWithManager_DisableTxOffloadingError(t *testing.T) {
	mgr := &LinkManagerMock{}
	mgr.On("AddLink", "br0", LinkTypeBridge).Return(nil)
	mgr.On("SetIP", "br0", mock.Anything, mock.Anything).Return(nil)
	mgr.On("BringUp", "br0").Return(nil)
	mgr.On("DisableTxOffloading", "br0").Return(errors.New("tx offload failed"))
	err := CreateBridgeWithManager(mgr, "br0", "192.168.1.0/24", true)
	require.EqualError(t, err, "tx offload failed")
}

func TestCreateBridgeWithManager_InvalidSubnet(t *testing.T) {
	mgr := &LinkManagerMock{}
	err := CreateBridgeWithManager(mgr, "br0", "invalid-subnet", false)
	require.Error(t, err)
}
