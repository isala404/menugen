import { useState, useEffect } from 'react'

// Get API URL from Choreo config or fallback to local development
const getApiUrl = () => {
  return window?.configs?.apiUrl || import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080'
}

function App() {
  const [user, setUser] = useState(null)
  const [isAuthenticated, setIsAuthenticated] = useState(false)
  const [isLoading, setIsLoading] = useState(true)
  const [selectedFile, setSelectedFile] = useState(null)
  const [menuId, setMenuId] = useState(null)
  const [menuData, setMenuData] = useState(null)
  const [isUploading, setIsUploading] = useState(false)
  const [error, setError] = useState(null)

  const BASE_URL = getApiUrl()

  // Check authentication status on component mount
  useEffect(() => {
    checkAuthStatus()
  }, [])

  const checkAuthStatus = async () => {
    try {
      const response = await fetch('/auth/userinfo', {
        credentials: 'include'
      })
      
      if (response.ok) {
        const userInfo = await response.json()
        setUser(userInfo)
        setIsAuthenticated(true)
      } else {
        setIsAuthenticated(false)
      }
    } catch (err) {
      setIsAuthenticated(false)
    } finally {
      setIsLoading(false)
    }
  }

  const handleLogin = () => {
    window.location.href = '/auth/login'
  }

  const handleLogout = () => {
    const sessionHint = document.cookie
      .split('; ')
      .find(row => row.startsWith('session_hint='))
      ?.split('=')[1]
    
    const logoutUrl = sessionHint 
      ? `/auth/logout?session_hint=${sessionHint}`
      : '/auth/logout'
    
    window.location.href = logoutUrl
  }

  const handleFileSelect = (event) => {
    const file = event.target.files[0]
    if (file) {
      // Validate file type
      if (!file.type.startsWith('image/')) {
        setError('Please select an image file')
        return
      }
      
      // Validate file size (8MB limit)
      if (file.size > 8 * 1024 * 1024) {
        setError('File size must be less than 8MB')
        return
      }
      
      setSelectedFile(file)
      setError(null)
      setMenuData(null)
      setMenuId(null)
    }
  }

  const handleUpload = async () => {
    if (!selectedFile) {
      setError('Please select a file first')
      return
    }

    setIsUploading(true)
    setError(null)

    try {
      const formData = new FormData()
      formData.append('image', selectedFile)

      const response = await fetch(`${BASE_URL}/api/menu`, {
        method: 'POST',
        credentials: 'include',
        body: formData,
      })

      if (!response.ok) {
        const errorData = await response.json()
        throw new Error(errorData.error?.message || 'Upload failed')
      }

      const data = await response.json()
      setMenuId(data.menu_id)
      
      // Start polling for menu status
      pollMenuStatus(data.menu_id)
    } catch (err) {
      setError(err.message)
    } finally {
      setIsUploading(false)
    }
  }

  const pollMenuStatus = async (id) => {
    try {
      const response = await fetch(`${BASE_URL}/api/menu/${id}`, {
        credentials: 'include'
      })
      
      if (!response.ok) {
        const errorData = await response.json()
        throw new Error(errorData.error?.message || 'Failed to fetch menu status')
      }

      const data = await response.json()
      setMenuData(data)

      // Continue polling if still processing
      if (data.status === 'PENDING' || data.status === 'PROCESSING') {
        setTimeout(() => pollMenuStatus(id), 2000) // Poll every 2 seconds
      }
    } catch (err) {
      setError(err.message)
    }
  }

  const resetForm = () => {
    setSelectedFile(null)
    setMenuId(null)
    setMenuData(null)
    setError(null)
    document.getElementById('file-input').value = ''
  }

  const getStatusColor = (status) => {
    switch (status) {
      case 'PENDING': return 'text-yellow-600'
      case 'PROCESSING': return 'text-blue-600'
      case 'COMPLETE': return 'text-green-600'
      case 'FAILED': return 'text-red-600'
      default: return 'text-gray-600'
    }
  }

  const getStatusIcon = (status) => {
    switch (status) {
      case 'PENDING': return '⏳'
      case 'PROCESSING': return '⚡'
      case 'COMPLETE': return '✅'
      case 'FAILED': return '❌'
      default: return '❓'
    }
  }

  const formatPrice = (priceCents, currency = 'USD') => {
    if (!priceCents) return 'Price not available'
    const dollars = priceCents / 100
    return new Intl.NumberFormat('en-US', {
      style: 'currency',
      currency: currency
    }).format(dollars)
  }

  const groupDishesBySection = (dishes, sections) => {
    const sectionMap = sections.reduce((acc, section) => {
      acc[section.id] = { ...section, dishes: [] }
      return acc
    }, {})

    // Add ungrouped section for dishes without section_id
    sectionMap['ungrouped'] = { id: 'ungrouped', name: 'Other Items', dishes: [] }

    dishes.forEach(dish => {
      const sectionId = dish.section_id || 'ungrouped'
      if (sectionMap[sectionId]) {
        sectionMap[sectionId].dishes.push(dish)
      }
    })

    // Filter out empty sections
    return Object.values(sectionMap).filter(section => section.dishes.length > 0)
  }

  // Show loading state while checking authentication
  if (isLoading) {
    return (
      <div className="min-h-screen bg-gray-50 flex items-center justify-center">
        <div className="text-center">
          <div className="text-4xl mb-4">⏳</div>
          <p className="text-lg text-gray-600">Loading...</p>
        </div>
      </div>
    )
  }

  // Show login screen if not authenticated
  if (!isAuthenticated) {
    return (
      <div className="min-h-screen bg-gray-50 flex items-center justify-center">
        <div className="max-w-md w-full bg-white rounded-lg shadow-lg p-8 text-center">
          <h1 className="text-3xl font-bold text-gray-900 mb-2">MenuGen</h1>
          <p className="text-lg text-gray-600 mb-8">Transform your menu photos into structured digital menus with AI</p>
          <button
            onClick={handleLogin}
            className="w-full px-6 py-3 bg-blue-600 text-white font-medium rounded-lg hover:bg-blue-700 transition-colors"
          >
            Sign In to Continue
          </button>
        </div>
      </div>
    )
  }

  return (
    <div className="min-h-screen bg-gray-50 py-8">
      <div className="max-w-6xl mx-auto px-4">
        {/* Header with logout */}
        <div className="text-center mb-8">
          <div className="flex justify-between items-center mb-4">
            <div></div>
            <div>
              <h1 className="text-4xl font-bold text-gray-900 mb-2">MenuGen</h1>
              <p className="text-xl text-gray-600">Transform your menu photos into structured digital menus with AI</p>
            </div>
            <div className="text-right">
              {user && (
                <div className="mb-2">
                  <p className="text-sm text-gray-600">Welcome, {user.preferred_username || user.email || 'User'}</p>
                </div>
              )}
              <button
                onClick={handleLogout}
                className="px-4 py-2 text-sm bg-gray-200 text-gray-700 rounded-lg hover:bg-gray-300 transition-colors"
              >
                Sign Out
              </button>
            </div>
          </div>
        </div>

        {/* Upload Section */}
        <div className="bg-white rounded-lg shadow-lg p-8 mb-8">
          <h2 className="text-2xl font-semibold text-gray-800 mb-6">Upload Menu Photo</h2>
          
          <div className="border-2 border-dashed border-gray-300 rounded-lg p-8 text-center">
            <input
              id="file-input"
              type="file"
              accept="image/*"
              onChange={handleFileSelect}
              className="hidden"
            />
            
            <label
              htmlFor="file-input"
              className="cursor-pointer flex flex-col items-center space-y-4"
            >
              <div className="text-6xl text-gray-400">📸</div>
              <div>
                <p className="text-lg font-medium text-gray-700">Click to select a menu photo</p>
                <p className="text-sm text-gray-500">Supports JPEG, PNG, WEBP (max 8MB)</p>
              </div>
            </label>
          </div>

          {selectedFile && (
            <div className="mt-4 p-4 bg-blue-50 rounded-lg">
              <p className="text-sm text-gray-700">
                <strong>Selected:</strong> {selectedFile.name} ({(selectedFile.size / 1024 / 1024).toFixed(2)} MB)
              </p>
            </div>
          )}

          {error && (
            <div className="mt-4 p-4 bg-red-50 border border-red-200 rounded-lg">
              <p className="text-red-700">{error}</p>
            </div>
          )}

          <div className="mt-6 flex space-x-4">
            <button
              onClick={handleUpload}
              disabled={!selectedFile || isUploading}
              className="px-6 py-3 bg-blue-600 text-white font-medium rounded-lg hover:bg-blue-700 disabled:bg-gray-400 disabled:cursor-not-allowed transition-colors"
            >
              {isUploading ? 'Uploading...' : 'Process Menu'}
            </button>
            
            <button
              onClick={resetForm}
              className="px-6 py-3 bg-gray-200 text-gray-700 font-medium rounded-lg hover:bg-gray-300 transition-colors"
            >
              Reset
            </button>
          </div>
        </div>

        {/* Status Section */}
        {menuData && (
          <div className="bg-white rounded-lg shadow-lg p-8 mb-8">
            <h2 className="text-2xl font-semibold text-gray-800 mb-6">Processing Status</h2>
            
            <div className="flex items-center space-x-4 mb-4">
              <span className="text-2xl">{getStatusIcon(menuData.status)}</span>
              <div>
                <p className={`text-lg font-medium ${getStatusColor(menuData.status)}`}>
                  {menuData.status}
                </p>
                <p className="text-sm text-gray-600">Menu ID: {menuData.menu_id}</p>
              </div>
            </div>

            {menuData.progress && (
              <div className="mb-4">
                <div className="flex justify-between items-center mb-2">
                  <span className="text-sm font-medium text-gray-700">Progress</span>
                  <span className="text-sm text-gray-600">
                    {menuData.progress.processed_dishes} / {menuData.progress.total_dishes} dishes
                  </span>
                </div>
                <div className="w-full bg-gray-200 rounded-full h-2">
                  <div
                    className="bg-blue-600 h-2 rounded-full transition-all duration-300"
                    style={{
                      width: `${(menuData.progress.processed_dishes / menuData.progress.total_dishes) * 100}%`
                    }}
                  ></div>
                </div>
              </div>
            )}

            {menuData.error && (
              <div className="p-4 bg-red-50 border border-red-200 rounded-lg">
                <p className="text-red-700">{menuData.error.message}</p>
              </div>
            )}
          </div>
        )}

        {/* Menu Display Section */}
        {menuData && menuData.status === 'COMPLETE' && menuData.menu && (
          <div className="bg-white rounded-lg shadow-lg p-8">
            <h2 className="text-2xl font-semibold text-gray-800 mb-6">Digital Menu</h2>
            
            {(() => {
              const groupedSections = groupDishesBySection(menuData.menu.dishes, menuData.menu.sections)
              
              return groupedSections.map((section) => (
                <div key={section.id} className="mb-8">
                  <h3 className="text-xl font-semibold text-gray-800 border-b-2 border-gray-200 pb-2 mb-4">
                    {section.name}
                  </h3>
                  
                  <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
                    {section.dishes.map((dish) => (
                      <div key={dish.id} className="border border-gray-200 rounded-lg p-4 hover:shadow-md transition-shadow">
                        {dish.image_url && (
                          <img
                            src={dish.image_url}
                            alt={dish.name}
                            className="w-full h-48 object-cover rounded-lg mb-3"
                            onError={(e) => {
                              e.target.style.display = 'none'
                            }}
                          />
                        )}
                        
                        <div className="space-y-2">
                          <h4 className="font-semibold text-gray-900">{dish.name}</h4>
                          
                          {dish.description && (
                            <p className="text-sm text-gray-600">{dish.description}</p>
                          )}
                          
                          <div className="flex justify-between items-center">
                            {dish.price_cents ? (
                              <span className="text-lg font-bold text-green-600">
                                {formatPrice(dish.price_cents, dish.currency)}
                              </span>
                            ) : dish.raw_price_string ? (
                              <span className="text-sm text-gray-600">{dish.raw_price_string}</span>
                            ) : null}
                            
                            <span className={`text-xs px-2 py-1 rounded-full ${
                              dish.status === 'COMPLETE' 
                                ? 'bg-green-100 text-green-800' 
                                : 'bg-yellow-100 text-yellow-800'
                            }`}>
                              {dish.status}
                            </span>
                          </div>
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              ))
            })()}
          </div>
        )}
      </div>
    </div>
  )
}

export default App
