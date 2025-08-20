import React from 'react';
import DishItem from './DishItem';

function MenuDisplay({ menu }) {
  if (!menu || !menu.dishes || menu.dishes.length === 0) {
    return (
      <div className="menu-container">
        <p>No menu data available.</p>
      </div>
    );
  }

  return (
    <div className="menu-container">
      <h2 className="menu-title">
        {menu.restaurant_name || 'Digital Menu'}
      </h2>
      <div className="dishes-grid">
        {menu.dishes.map((dish, index) => (
          <DishItem key={index} dish={dish} />
        ))}
      </div>
    </div>
  );
}

export default MenuDisplay;
