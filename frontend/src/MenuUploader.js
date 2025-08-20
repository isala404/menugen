import React, { useState, useRef } from 'react';
import axios from 'axios';

function MenuUploader({ onMenuProcessed, onStatusUpdate }) {
  const [selectedFile, setSelectedFile] = useState(null);
  const [uploading, setUploading] = useState(false);
  const [dragOver, setDragOver] = useState(false);
  const fileInputRef = useRef(null);

  // Get API URL from Choreo injected config or fallback to localhost for development
  const getApiUrl = () => {
    if (window.configs && window.configs.apiUrl) {
      return window.configs.apiUrl;
    }
    return 'http://localhost:8080'; // Fallback for local development
  };

  const handleFileSelect = (event) => {
    const file = event.target.files[0];
    if (file && file.type.startsWith('image/')) {
      setSelectedFile(file);
      onStatusUpdate('Image selected. Ready to upload.');
    } else {
      onStatusUpdate('Error: Please select a valid image file.');
    }
  };

  const handleDrop = (event) => {
    event.preventDefault();
    setDragOver(false);
    
    const file = event.dataTransfer.files[0];
    if (file && file.type.startsWith('image/')) {
      setSelectedFile(file);
      onStatusUpdate('Image selected. Ready to upload.');
    } else {
      onStatusUpdate('Error: Please select a valid image file.');
    }
  };

  const handleDragOver = (event) => {
    event.preventDefault();
    setDragOver(true);
  };

  const handleDragLeave = (event) => {
    event.preventDefault();
    setDragOver(false);
  };

  const handleUpload = async () => {
    if (!selectedFile) {
      onStatusUpdate('Error: Please select an image first.');
      return;
    }

    setUploading(true);
    onStatusUpdate('Uploading image and processing menu...');

    const formData = new FormData();
    formData.append('image', selectedFile);

    try {
      const response = await axios.post(`${getApiUrl()}/api/menu`, formData, {
        headers: {
          'Content-Type': 'multipart/form-data',
        },
        timeout: 120000, // 2 minute timeout for processing
      });

      if (response.data.success) {
        onMenuProcessed(response.data.menu);
        onStatusUpdate('Menu processed successfully!');
        setSelectedFile(null);
        if (fileInputRef.current) {
          fileInputRef.current.value = '';
        }
      } else {
        onStatusUpdate('Error: Failed to process menu.');
      }
    } catch (error) {
      console.error('Upload error:', error);
      if (error.code === 'ECONNABORTED') {
        onStatusUpdate('Error: Request timed out. Please try again.');
      } else if (error.response) {
        onStatusUpdate(`Error: ${error.response.data.message || 'Failed to process menu.'}`);
      } else {
        onStatusUpdate('Error: Unable to connect to server.');
      }
    } finally {
      setUploading(false);
    }
  };

  return (
    <div className="upload-container">
      <div 
        className={`upload-area ${dragOver ? 'dragover' : ''}`}
        onDrop={handleDrop}
        onDragOver={handleDragOver}
        onDragLeave={handleDragLeave}
        onClick={() => fileInputRef.current?.click()}
      >
        <div className="upload-icon">📸</div>
        <div className="upload-text">
          {selectedFile ? selectedFile.name : 'Drop your menu image here or click to browse'}
        </div>
        <input
          ref={fileInputRef}
          type="file"
          accept="image/*"
          onChange={handleFileSelect}
          className="file-input"
        />
      </div>
      
      <button 
        className="upload-button"
        onClick={handleUpload}
        disabled={!selectedFile || uploading}
      >
        {uploading ? (
          <>
            <div className="loading-spinner"></div>
            Processing Menu...
          </>
        ) : (
          'Upload & Process Menu'
        )}
      </button>
    </div>
  );
}

export default MenuUploader;
