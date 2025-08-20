import React from 'react';

function DishItem({ dish }) {
  return (
    <div className="dish-item">
      {dish.image_url && (
        <img
          src={dish.image_url}
          alt={dish.name}
          className="dish-image"
          onError={(e) => {
            e.target.style.display = 'none';
          }}
        />
      )}
      <div className="dish-content">
        <h3 className="dish-name">{dish.name}</h3>
        {dish.description && (
          <p className="dish-description">{dish.description}</p>
        )}
        {dish.price && (
          <div className="dish-price">{dish.price}</div>
        )}
      </div>
    </div>
  );
}

export default DishItem;
